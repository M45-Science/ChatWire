// Package firewall provides the shared moderator firewall service.
package firewall

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"context"
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

func runUFW(mutating bool, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := os.Stat(ufwPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("UFW is not installed at %s", ufwPath)
		}
		return "", fmt.Errorf("unable to check UFW availability: %w", err)
	}

	fullArgs := append([]string(nil), args...)

	runCmd := exec.CommandContext(ctx, ufwPath, fullArgs...)
	if os.Geteuid() != 0 {
		if _, err := exec.LookPath("sudo"); err != nil {
			return "", errors.New("sudo is not installed, so ChatWire cannot run UFW")
		}
		runCmd = exec.CommandContext(ctx, "sudo", append([]string{"-n", ufwPath}, fullArgs...)...)
	}
	if mutating && len(fullArgs) > 0 && strings.EqualFold(fullArgs[0], "delete") {
		runCmd.Stdin = strings.NewReader("y\n")
	}

	output, err := runCmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
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

var actionMu sync.Mutex

func List() ([]string, error) {
	actionMu.Lock()
	defer actionMu.Unlock()
	return getCurrentUFWDenyIPs()
}
func Execute(action, rawIP, actor string) (string, error) {
	actionMu.Lock()
	defer actionMu.Unlock()
	if action == "list" {
		ips, e := getCurrentUFWDenyIPs()
		return strings.Join(ips, "\n"), e
	}
	if action != "ban" && action != "unban" {
		return "", errors.New("choose ban, unban, or list")
	}
	ip, msg := validatePublicIPv4(sanitizeIPInput(rawIP))
	if msg != "" {
		return "", errors.New(msg)
	}
	if wait, limited := firewallLimiter.Check(); limited {
		return "", fmt.Errorf("firewall commands are cooling down; retry in %s", wait.Round(time.Second))
	}
	args := []string{"deny", "from", ip.String()}
	if action == "unban" {
		ips, e := getCurrentUFWDenyIPs()
		if e != nil {
			return "", errors.New("unable to read firewall rules")
		}
		if !slices.Contains(ips, ip.String()) {
			return "", errors.New("IP has no deny rule")
		}
		args = append([]string{"delete"}, args...)
	}
	out, e := runUFW(true, args...)
	if e != nil {
		cwlog.DoLogCW("UFW operation failed: %v", e)
		return "", errors.New("UFW command failed; check passwordless sudo and instance logs")
	}
	firewallLimiter.RecordSuccess()
	cwlog.DoLogAudit("IP-%s actor=%s ip=%s", action, actor, ip.String())
	return out, nil
}
