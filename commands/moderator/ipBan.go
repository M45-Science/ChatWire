package moderator

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

const ufwPath = "/usr/sbin/ufw"

const (
	firewallBurstGrace  = 4
	firewallResetWindow = 15 * time.Minute
	firewallBaseDelay   = 10 * time.Second
	firewallMaxDelay    = 15 * time.Minute
)

var blockedIPv4Ranges = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
}

var blockedIPv4Nets = mustParseCIDRs(blockedIPv4Ranges)

var firewallLimiter = firewallRateLimiter{}

type firewallRateLimiter struct {
	mu          sync.Mutex
	streak      int
	lastAction  time.Time
	nextAllowed time.Time
}

// IPBan manages UFW deny rules for validated public IPv4 addresses.
func IPBan(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	var action string
	var rawIP string

	a := i.ApplicationCommandData()
	for _, arg := range a.Options {
		if arg.Type != discordgo.ApplicationCommandOptionString {
			continue
		}

		switch strings.ToLower(arg.Name) {
		case "action":
			action = strings.ToLower(strings.TrimSpace(arg.StringValue()))
		case "ip":
			rawIP = sanitizeIPInput(arg.StringValue())
		}
	}

	switch action {
	case "list":
		handleFirewallList(i)
	case "ban":
		handleFirewallBan(i, rawIP)
	case "unban":
		handleFirewallUnban(i, rawIP)
	default:
		cwlog.DoLogAudit("IP-COMMAND INVALID: %v: action=%q ip=%q", i.Member.User.Username, action, rawIP)
		disc.InteractionEphemeralResponse(i, "Error:", "You must choose an action: ban, unban, or list.")
	}
}

func handleFirewallBan(i *discordgo.InteractionCreate, rawIP string) {
	ipv4, msg := validatePublicIPv4(rawIP)
	if msg != "" {
		cwlog.DoLogAudit("IP-BAN REJECTED: %v: %q: %v", i.Member.User.Username, rawIP, msg)
		disc.InteractionEphemeralResponse(i, "Error:", msg)
		return
	}

	if wait, limited := firewallLimiter.Check(); limited {
		cwlog.DoLogAudit("IP-BAN RATE-LIMITED: %v: %v: wait=%v", i.Member.User.Username, ipv4.String(), wait.Round(time.Second))
		disc.InteractionEphemeralResponse(i, "Error:", fmt.Sprintf("Firewall commands are cooling down. Try again in about %s.", wait.Round(time.Second)))
		return
	}

	ipText := ipv4.String()
	output, err := runUFW(true, "deny", "from", ipText)
	if err != nil {
		handleUFWError(i, "IP-BAN FAILED", ipText, err, output)
		return
	}

	firewallLimiter.RecordSuccess()
	cwlog.DoLogAudit("IP-BAN: %v: %v", i.Member.User.Username, ipText)
	if output == "" {
		output = "UFW deny rule added."
	}
	disc.InteractionEphemeralResponse(i, "Complete:", output)
}

func handleFirewallUnban(i *discordgo.InteractionCreate, rawIP string) {
	ipv4, msg := validatePublicIPv4(rawIP)
	if msg != "" {
		cwlog.DoLogAudit("IP-UNBAN REJECTED: %v: %q: %v", i.Member.User.Username, rawIP, msg)
		disc.InteractionEphemeralResponse(i, "Error:", msg)
		return
	}

	if wait, limited := firewallLimiter.Check(); limited {
		cwlog.DoLogAudit("IP-UNBAN RATE-LIMITED: %v: %v: wait=%v", i.Member.User.Username, ipv4.String(), wait.Round(time.Second))
		disc.InteractionEphemeralResponse(i, "Error:", fmt.Sprintf("Firewall commands are cooling down. Try again in about %s.", wait.Round(time.Second)))
		return
	}

	ipText := ipv4.String()
	denyIPs, err := getCurrentUFWDenyIPs()
	if err != nil {
		handleUFWError(i, "IP-UNBAN LIST FAILED", ipText, err, "")
		return
	}
	if !slices.Contains(denyIPs, ipText) {
		cwlog.DoLogAudit("IP-UNBAN REJECTED: %v: %v: no DENY rule present", i.Member.User.Username, ipText)
		disc.InteractionEphemeralResponse(i, "Error:", "That IP does not currently have a UFW DENY rule.")
		return
	}

	output, err := runUFW(true, "delete", "deny", "from", ipText)
	if err != nil {
		handleUFWError(i, "IP-UNBAN FAILED", ipText, err, output)
		return
	}

	firewallLimiter.RecordSuccess()
	cwlog.DoLogAudit("IP-UNBAN: %v: %v", i.Member.User.Username, ipText)
	if output == "" {
		output = "UFW deny rule removed."
	}
	disc.InteractionEphemeralResponse(i, "Complete:", output)
}

func handleFirewallList(i *discordgo.InteractionCreate) {
	denyIPs, err := getCurrentUFWDenyIPs()
	if err != nil {
		handleUFWError(i, "IP-LIST FAILED", "", err, "")
		return
	}

	cwlog.DoLogAudit("IP-LIST: %v: %v entries", i.Member.User.Username, len(denyIPs))

	if len(denyIPs) == 0 {
		disc.InteractionEphemeralResponse(i, "UFW DENY List", "No matching IPv4 DENY rules were found.")
		return
	}

	buf := strings.Join(denyIPs, "\n")
	disc.InteractionEphemeralResponse(i, "UFW DENY List", buf)
}

func runUFW(mutating bool, args ...string) (string, error) {
	if _, err := os.Stat(ufwPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("UFW is not installed at %s", ufwPath)
		}
		return "", fmt.Errorf("unable to check UFW availability: %w", err)
	}

	fullArgs := append([]string(nil), args...)

	runCmd := exec.Command(ufwPath, fullArgs...)
	if os.Geteuid() != 0 {
		if _, err := exec.LookPath("sudo"); err != nil {
			return "", errors.New("sudo is not installed, so ChatWire cannot run UFW")
		}
		runCmd = exec.Command("sudo", append([]string{"-n", ufwPath}, fullArgs...)...)
	}
	if mutating && len(fullArgs) > 0 && strings.EqualFold(fullArgs[0], "delete") {
		runCmd.Stdin = strings.NewReader("y\n")
	}

	output, err := runCmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func handleUFWError(i *discordgo.InteractionCreate, auditPrefix, ipText string, err error, output string) {
	if ipText != "" {
		cwlog.DoLogAudit("%s: %v: %v: %v", auditPrefix, i.Member.User.Username, ipText, err)
		cwlog.DoLogCW("%s for %s: %v: %s", auditPrefix, ipText, err, output)
	} else {
		cwlog.DoLogAudit("%s: %v: %v", auditPrefix, i.Member.User.Username, err)
		cwlog.DoLogCW("%s: %v: %s", auditPrefix, err, output)
	}

	bufLower := strings.ToLower(output)
	switch {
	case strings.Contains(bufLower, "password is required"), strings.Contains(bufLower, "a password is required"):
		disc.InteractionEphemeralResponse(i, "Error:", "This command needs passwordless sudo access to /usr/sbin/ufw.")
		return
	case errors.Is(err, exec.ErrNotFound):
		disc.InteractionEphemeralResponse(i, "Error:", err.Error())
		return
	}

	msg := "UFW command failed."
	if err != nil && output == "" {
		msg = err.Error()
	} else if output != "" {
		msg = output
	}
	disc.InteractionEphemeralResponseColor(i, "Error:", msg, glob.COLOR_RED)
}

func getCurrentUFWDenyIPs() ([]string, error) {
	output, err := runUFW(false, "status")
	if err != nil {
		return nil, err
	}
	if strings.Contains(strings.ToLower(output), "status: inactive") {
		return nil, errors.New("UFW is inactive")
	}

	var denyIPs []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		if !strings.EqualFold(fields[1], "DENY") {
			continue
		}

		ip := net.ParseIP(fields[2])
		if ip == nil || ip.To4() == nil {
			continue
		}
		denyIPs = append(denyIPs, ip.String())
	}

	slices.Sort(denyIPs)
	return slices.Compact(denyIPs), nil
}

func sanitizeIPInput(input string) string {
	input = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, input)
	return strings.TrimSpace(input)
}

func validatePublicIPv4(input string) (net.IP, string) {
	if input == "" {
		return nil, "You must supply an IP address."
	}

	ip := net.ParseIP(input)
	if ip == nil {
		return nil, "That is not a valid IP address."
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return nil, "Only IPv4 addresses are allowed."
	}

	if isBlockedIPv4(ipv4) {
		return nil, "Only public IPv4 addresses are allowed."
	}

	localPublicIPs, err := getLocalPublicIPv4s()
	if err != nil {
		cwlog.DoLogCW("IPBan: unable to enumerate local IPs: %v", err)
		return nil, "Unable to verify the server's own IP addresses."
	}
	if slices.Contains(localPublicIPs, ipv4.String()) {
		return nil, "Refusing to ban this server's own IP address."
	}

	domainPublicIPs, err := getProtectedDomainIPv4s()
	if err != nil {
		cwlog.DoLogCW("IPBan: unable to resolve protected domains: %v", err)
		return nil, "Unable to verify protected public domain IP addresses."
	}
	if slices.Contains(domainPublicIPs, ipv4.String()) {
		return nil, "Refusing to ban a protected public domain IP address."
	}

	return ipv4, ""
}

func getLocalPublicIPv4s() ([]string, error) {
	var out []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet == nil {
			continue
		}

		ipv4 := ipNet.IP.To4()
		if ipv4 == nil || isBlockedIPv4(ipv4) {
			continue
		}
		out = append(out, ipv4.String())
	}

	slices.Sort(out)
	return slices.Compact(out), nil
}

func getProtectedDomainIPv4s() ([]string, error) {
	var out []string

	protectedDomains := []string{"m45sci.xyz"}
	if domain := strings.TrimSpace(cfg.Global.Paths.URLs.Domain); domain != "" && !strings.EqualFold(domain, "localhost") {
		protectedDomains = append(protectedDomains, domain)
	}

	for _, domain := range protectedDomains {
		ips, err := net.LookupIP(domain)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			ipv4 := ip.To4()
			if ipv4 == nil || isBlockedIPv4(ipv4) {
				continue
			}
			out = append(out, ipv4.String())
		}
	}

	slices.Sort(out)
	return slices.Compact(out), nil
}

func isBlockedIPv4(ip net.IP) bool {
	for _, blocked := range blockedIPv4Nets {
		if blocked.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			cwlog.DoLogCW("Skipping invalid blocked IPv4 CIDR %q: %v", cidr, err)
			continue
		}
		out = append(out, ipNet)
	}
	return out
}

func (r *firewallRateLimiter) Check() (time.Duration, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Before(r.nextAllowed) {
		return r.nextAllowed.Sub(now), true
	}

	return 0, false
}

func (r *firewallRateLimiter) RecordSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if r.lastAction.IsZero() || now.Sub(r.lastAction) > firewallResetWindow {
		r.streak = 0
		r.nextAllowed = time.Time{}
	}

	r.streak++
	r.lastAction = now

	if r.streak <= firewallBurstGrace {
		r.nextAllowed = time.Time{}
		return
	}

	shift := r.streak - firewallBurstGrace - 1
	delay := firewallBaseDelay
	if shift > 0 {
		delay = firewallBaseDelay * time.Duration(1<<shift)
	}
	if delay > firewallMaxDelay {
		delay = firewallMaxDelay
	}

	r.nextAllowed = now.Add(delay)
}
