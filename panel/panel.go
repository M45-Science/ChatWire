package panel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"math/big"
	"net"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/M45-Science/rcon"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands"
	"ChatWire/commands/moderator"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/support"
	"ChatWire/watcher"

	"github.com/hako/durafmt"
)

var (
	panelTmpl     *template.Template
	panelTmplLock sync.RWMutex
)

type panelCmd struct {
	Cmd   string
	Label string
	Icon  string
}

type panelSave struct {
	Name string
	Age  string
}

type panelCommand struct {
	Name        string
	Description string
}

type panelCmdGroup struct {
	Name string
	Cmds []panelCmd
}

func loadTemplate() {
	tmpl, err := template.ParseFiles(constants.PanelTemplateFile)
	if err != nil {
		cwlog.DoLogCW("panel: template load error: %v", err)
		return
	}
	panelTmplLock.Lock()
	panelTmpl = tmpl
	panelTmplLock.Unlock()
}

// WatchTemplate monitors the panel template and reloads it when changed.
func WatchTemplate() {
	time.Sleep(time.Second)
	loadTemplate()
	watcher.Watch(constants.PanelTemplateFile, 5*time.Second, &glob.ServerRunning, loadTemplate)
}

var modCmdGroups = []panelCmdGroup{
	{
		Name: "ChatWire",
		Cmds: []panelCmd{
			{Cmd: "reboot-chatwire", Label: "Reboot ChatWire", Icon: "restart_alt"},
			{Cmd: "queue-reboot", Label: "Queue ChatWire Reboot", Icon: "schedule"},
			{Cmd: "force-reboot", Label: "Force ChatWire Reboot", Icon: "restart_alt"},
			{Cmd: "queue-fact-reboot", Label: "Queue Factorio Reboot", Icon: "schedule"},
			{Cmd: "reload-config", Label: "Reload Config", Icon: "refresh"},
		},
	},
	{
		Name: "Factorio",
		Cmds: []panelCmd{
			{Cmd: "start-factorio", Label: "Start Factorio", Icon: "play_arrow"},
			{Cmd: "stop-factorio", Label: "Stop Factorio", Icon: "stop"},
			{Cmd: "install-factorio", Label: "Install Factorio", Icon: "download"},
			{Cmd: "update-factorio", Label: "Update Factorio", Icon: "update"},
			{Cmd: "new-map", Label: "New Map", Icon: "create_new_folder"},
			{Cmd: "archive-map", Label: "Archive Map", Icon: "archive"},
			{Cmd: "map-reset", Label: "Reset Map", Icon: "map"},
		},
	},
	{
		Name: "Mods",
		Cmds: []panelCmd{
			{Cmd: "update-mods", Label: "Update Mods", Icon: "system_update_alt"},
			{Cmd: "sync-mods", Label: "Sync Mods", Icon: "sync"},
		},
	},
}

type panelData struct {
	ServerName    string
	Callsign      string
	CWVersion     string
	Factorio      string
	SoftMod       string
	Players       int
	Gametime      string
	SaveName      string
	UPS           string
	CWUp          string
	FactUp        string
	NextReset     string
	TimeTill      string
	ResetInterval string
	Total         int
	Mods          int
	Banned        int
	Mem           int
	Reg           int
	Vet           int
	AccessLevel   int
	ModNames      []string
	PlayHours     string
	HoursEnabled  bool
	Paused        bool
	FactRunning   bool
	MapSchedule   bool
	Token         string
	CmdGroups     []panelCmdGroup
	Saves         []panelSave
	Commands      []panelCommand
	Info          string
	LocalCfg      string
	GlobalCfg     string
	LocalJSON     string
}

// Start runs the HTTPS status panel server.
func Start() {
	http.HandleFunc("/panel", handlePanel)
	http.HandleFunc("/panel-data", handlePanelData)
	http.HandleFunc("/action", handleAction)
	go WatchTemplate()
	addr := fmt.Sprintf(":%v", cfg.Local.Port+constants.PanelPortOffset)
	go func() {
		cert, err := generateCert()
		if err != nil {
			cwlog.DoLogCW("Panel TLS error: %v", err)
			return
		}
		srv := &http.Server{Addr: addr, TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}}}
		cwlog.DoLogCW("Panel server listening on %v", addr)
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			cwlog.DoLogCW("Panel server error: %v", err)
		}
	}()
}

// GenerateToken creates a temporary token for web access.
func GenerateToken(id string) string {
	now := time.Now().Unix()
	token := glob.RandomBase64String(128)
	var orig int64 = now
	glob.PanelTokenLock.Lock()
	for k, v := range glob.PanelTokens {
		if v.DiscID == id {
			if v.Orig < orig {
				orig = v.Orig
			}
			delete(glob.PanelTokens, k)
		}
	}
	if now-orig > constants.PanelTokenLimitSec {
		orig = now
	}
	glob.PanelTokens[token] = &glob.PanelTokenData{Token: token, DiscID: id, Time: now, Orig: orig, IP: ""}
	glob.PanelTokenLock.Unlock()
	return token
}

func tokenValid(tok string) bool {
	glob.PanelTokenLock.RLock()
	data, ok := glob.PanelTokens[tok]
	glob.PanelTokenLock.RUnlock()
	if !ok {
		return false
	}
	now := time.Now().Unix()
	if now-data.Time > constants.PassExpireSec {
		return false
	}
	if now-data.Orig > constants.PanelTokenLimitSec {
		return false
	}
	return true
}

func buildPanelData(tok string) panelData {
	cwUptime := time.Since(glob.Uptime.Round(time.Second)).Round(time.Second).String()

	factUptime := "not running"
	if !fact.FactorioBootedAt.IsZero() && fact.FactorioBooted {
		factUptime = time.Since(fact.FactorioBootedAt.Round(time.Second)).Round(time.Second).String()
	}
	factRunning := fact.FactIsRunning

	nextReset := ""
	timeTill := ""
	resetInterval := ""
	if fact.HasResetTime() {
		nextReset = fact.FormatResetTime()
		timeTill = fact.TimeTillReset()
		resetInterval = fact.FormatResetInterval()
	}
	mapSchedule := fact.HasResetTime() || fact.HasResetInterval()

	ups := "no data"
	if ten, thirty, hour := fact.GetFactUPS(); ten > 0 || thirty > 0 || hour > 0 {
		ups = fmt.Sprintf("%.2f/%.2f/%.2f", ten, thirty, hour)
	}

	playHours := ""
	if cfg.Local.Options.PlayHourEnable {
		playHours = fmt.Sprintf("%d-%d GMT", cfg.Local.Options.PlayStartHour, cfg.Local.Options.PlayEndHour)
	}

	accessLevel := 0
	if cfg.Local.Options.RegularsOnly {
		accessLevel = 2
	} else if cfg.Local.Options.MembersOnly {
		accessLevel = 1
	}

	paused := false
	if fact.PausedTicks > 4 {
		paused = true
	}

	var mem, reg, vet, mods, ban int
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		switch player.Level {
		case -1:
			ban++
		case 1:
			mem++
		case 2:
			reg++
		case 3:
			vet++
		case 255:
			mods++
		}
	}
	bCount := len(banlist.BanList)
	ban += bCount
	glob.PlayerListLock.RUnlock()
	total := ban + mem + reg + vet + mods

	softMod := ""
	if glob.SoftModVersion != constants.Unknown {
		softMod = glob.SoftModVersion
	}

	var saves []panelSave
	files, err := os.ReadDir(cfg.GetSavesFolder())
	if err == nil {
		var entries []os.DirEntry
		for _, f := range files {
			name := f.Name()
			if strings.HasSuffix(name, ".zip") && !strings.HasSuffix(name, "tmp.zip") && !strings.HasSuffix(name, cfg.Local.Name+"_new.zip") {
				entries = append(entries, f)
			}
		}
		sort.Slice(entries, func(i, j int) bool {
			iInfo, _ := entries[i].Info()
			jInfo, _ := entries[j].Info()
			return iInfo.ModTime().After(jInfo.ModTime())
		})
		if len(entries) > 24 {
			entries = entries[:24]
		}
		for _, f := range entries {
			info, _ := f.Info()
			modAge := time.Since(info.ModTime()).Round(time.Second)
			units, _ := durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,us:us")
			ageStr := durafmt.Parse(modAge).LimitFirstN(2).Format(units) + " ago"
			saves = append(saves, panelSave{Name: strings.TrimSuffix(f.Name(), ".zip"), Age: ageStr})
		}
	}
	if len(saves) > 16 {
		saves = saves[:16]
	}

	var cmdList []panelCommand
	for _, c := range commands.ListAllCommands() {
		cmdList = append(cmdList, panelCommand{Name: "/" + c.AppCmd.Name, Description: c.AppCmd.Description})
	}
	sort.Slice(cmdList, func(i, j int) bool { return cmdList[i].Name < cmdList[j].Name })

	groups := make([]panelCmdGroup, len(modCmdGroups))
	copy(groups, modCmdGroups)
	for idx := range groups {
		sort.Slice(groups[idx].Cmds, func(i, j int) bool { return groups[idx].Cmds[i].Label < groups[idx].Cmds[j].Label })
	}
	modFiles, _ := modupdate.GetModFiles()
	var modNames []string
	for _, m := range modFiles {
		modNames = append(modNames, fmt.Sprintf("%s (%s)", m.Name, m.Version))
	}
	sort.Strings(modNames)
	skip := map[string]struct{}{
		"Next map reset":                   {},
		"Reset interval":                   {},
		"Map Reset Hour":                   {},
		"Skip Map Reset":                   {},
		"Limit Open Hours":                 {},
		"Open Hour":                        {},
		"Close Hour":                       {},
		"Callsign":                         {},
		"Port":                             {},
		"Channel ID":                       {},
		"Channel":                          {},
		"Last Backup Slot":                 {},
		"Regulars, Veterans only":          {},
		"Members, Regulars, Veterans only": {},
	}
	pd := panelData{ServerName: cfg.Local.Name, Callsign: strings.ToUpper(cfg.Local.Callsign),
		CWVersion: constants.Version, Factorio: fact.FactorioVersion, SoftMod: softMod,
		Players: fact.NumPlayers, Gametime: fact.GametimeString, SaveName: fact.LastSaveName,
		CWUp: cwUptime, FactUp: factUptime, UPS: ups,
		NextReset: nextReset, TimeTill: timeTill, ResetInterval: resetInterval,
		Total: total, Mods: mods, Banned: ban, PlayHours: playHours, Paused: paused,
		FactRunning: factRunning, MapSchedule: mapSchedule,
		Token: tok, CmdGroups: groups, Saves: saves, Commands: cmdList, Info: buildInfoString(),
		ModNames:    modNames,
		AccessLevel: accessLevel,
		LocalCfg: func() string {
			s := buildCfgStringSkip(cfg.Local, skip)
			switch accessLevel {
			case 2:
				s += "\nMinimum Level: Regulars+"
			case 1:
				s += "\nMinimum Level: Members+"
			default:
				s += "\nMinimum Level: Open"
			}
			return s
		}(),
		GlobalCfg: buildCfgString(cfg.Global),
		LocalJSON: func() string {
			b, _ := json.MarshalIndent(cfg.Local, "", "  ")
			return string(b)
		}()}
	return pd
}

func handlePanel(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	if tok == "" || !tokenValid(tok) {
		if !*glob.LocalTestMode {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
	}
	glob.PanelTokenLock.Lock()
	data, ok := glob.PanelTokens[tok]
	if ok {
		if data.IP == "" {
			data.IP = ip
		} else if data.IP != ip {
			delete(glob.PanelTokens, tok)
			ok = false
		}
	}
	glob.PanelTokenLock.Unlock()
	if !ok && !*glob.LocalTestMode {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	panelTmplLock.RLock()
	t := panelTmpl
	panelTmplLock.RUnlock()
	if t == nil {
		loadTemplate()
		panelTmplLock.RLock()
		t = panelTmpl
		panelTmplLock.RUnlock()
		if t == nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
	}

	pd := buildPanelData(tok)
	_ = t.Execute(w, pd)
}

func handlePanelData(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	if tok == "" || !tokenValid(tok) {
		if !*glob.LocalTestMode {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
	}
	glob.PanelTokenLock.Lock()
	data, ok := glob.PanelTokens[tok]
	if ok {
		if data.IP == "" {
			data.IP = ip
		} else if data.IP != ip {
			delete(glob.PanelTokens, tok)
			ok = false
		}
	}
	glob.PanelTokenLock.Unlock()
	if !ok && !*glob.LocalTestMode {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	pd := buildPanelData(tok)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pd)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tok := r.FormValue("token")
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	cmd := r.FormValue("cmd")
	if cmd == "" || (tok == "" && !*glob.LocalTestMode) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	glob.PanelTokenLock.Lock()
	userInfo, ok := glob.PanelTokens[tok]
	if ok {
		if userInfo.IP == "" {
			userInfo.IP = ip
		} else if userInfo.IP != ip {
			delete(glob.PanelTokens, tok)
			ok = false
		}
	}
	glob.PanelTokenLock.Unlock()
	if !ok || !tokenValid(tok) {
		if !*glob.LocalTestMode {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		userInfo = &glob.PanelTokenData{Name: "local"}
	}

	switch cmd {
	case "force-reboot":
		moderator.ForceReboot(nil, nil)
	case "queue-reboot":
		moderator.QueReboot(nil, nil)
	case "queue-fact-reboot":
		moderator.QueFactReboot(nil, nil)
	case "reboot-chatwire":
		moderator.RebootCW(nil, nil)
	case "reload-config":
		moderator.ReloadConfig(nil, nil)
	case "start-factorio":
		moderator.StartFact(nil, nil)
	case "stop-factorio":
		moderator.StopFact(nil, nil)
	case "new-map":
		moderator.NewMap(nil, nil)
	case "archive-map":
		moderator.ArchiveMap(nil, nil)
	case "update-mods":
		moderator.UpdateMods(nil, nil)
	case "sync-mods":
		moderator.SyncMods(nil, nil)
	case "update-factorio":
		moderator.UpdateFactorio(nil, nil)
	case "install-factorio":
		moderator.InstallFactorio(nil, nil)
	case "map-reset":
		moderator.MapReset(nil, nil)
	case "change-map":
		arg := r.FormValue("arg")
		if arg == "" {
			fmt.Fprint(w, "map name required")
			return
		}
		fact.DoChangeMap(arg)
	case "rcon":
		cmdStr := r.FormValue("arg")
		if cmdStr == "" {
			fmt.Fprint(w, "command required")
			return
		}
		cmdStr = strings.TrimPrefix(cmdStr, "/")
		cmdStr = "/" + cmdStr
		portstr := fmt.Sprintf("%v", cfg.Local.Port+cfg.Global.Options.RconOffset)
		rc, err := rcon.Dial("localhost"+":"+portstr, glob.RCONPass)
		if err != nil || rc == nil {
			fmt.Fprint(w, "RCON connect failed")
			return
		}
		reqID, err := rc.Write(cmdStr)
		if err != nil {
			fmt.Fprint(w, "RCON write failed")
			return
		}
		resp, respID, err := rc.Read()
		if err != nil || reqID != respID {
			fmt.Fprint(w, "RCON read failed")
			return
		}
		if resp == "" {
			resp = "(Empty response)"
		}
		fmt.Fprint(w, resp)
		cwlog.DoLogAudit("%v: rcon %s", userInfo.Name, cmdStr)
		return
	case "config-hours":
		if r.FormValue("enabled") != "" {
			cfg.Local.Options.PlayHourEnable = true
		} else {
			cfg.Local.Options.PlayHourEnable = false
		}
		if v := r.FormValue("start"); v != "" {
			if val, err := strconv.Atoi(v); err == nil && val >= 0 && val < 24 {
				cfg.Local.Options.PlayStartHour = val
			}
		}
		if v := r.FormValue("end"); v != "" {
			if val, err := strconv.Atoi(v); err == nil && val >= 0 && val < 24 {
				cfg.Local.Options.PlayEndHour = val
			}
		}
		cfg.WriteLCfg()
	case "set-schedule":
		n := cfg.ResetInterval{}
		if v := r.FormValue("months"); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				n.Months = val
			}
		}
		if v := r.FormValue("weeks"); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				n.Weeks = val
			}
		}
		if v := r.FormValue("days"); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				n.Days = val
			}
		}
		if v := r.FormValue("hours"); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				n.Hours = val
			}
		}
		cfg.Local.Options.ResetInterval = n
		fact.SetResetDate()
		if v := r.FormValue("date"); v != "" {
			layout := "2006-01-02T15:04"
			if t, err := time.Parse(layout, v); err == nil && t.After(time.Now().UTC()) {
				cfg.Local.Options.NextReset = t
			}
		}
		if cfg.Local.Options.NextReset.UTC().Sub(time.Now().UTC()) > (time.Hour*24*90 + time.Hour*24) {
			cfg.Local.Options.NextReset = time.Time{}
		}
		cfg.WriteLCfg()
		support.ConfigSoftMod()
	case "disable-schedule":
		cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
		cfg.Local.Options.NextReset = time.Time{}
		cfg.WriteLCfg()
		support.ConfigSoftMod()
	case "apply-config":
		text := r.FormValue("cfg")
		if text == "" {
			fmt.Fprint(w, "no config provided")
			return
		}
		newCfg := cfg.Local
		if err := json.Unmarshal([]byte(text), &newCfg); err != nil {
			fmt.Fprint(w, "parse error")
			return
		}
		cfg.Local = newCfg
		cfg.WriteLCfg()
		support.ConfigSoftMod()
	case "set-config-field":
		path := r.FormValue("path")
		val := r.FormValue("value")
		if path == "" {
			fmt.Fprint(w, "path required")
			return
		}
		if err := setCfgField(strings.Split(path, "."), val); err != nil {
			fmt.Fprint(w, err.Error())
			return
		}
		cfg.WriteLCfg()
		support.ConfigSoftMod()
	case "discord":
		arg := strings.TrimPrefix(r.FormValue("arg"), "/")
		for _, c := range commands.ListAllCommands() {
			if strings.EqualFold(c.AppCmd.Name, arg) {
				c.Function(&c, fakeInteraction(userInfo))
				fmt.Fprintf(w, "discord command '%s' executed", arg)
				cwlog.DoLogAudit("%v: discord %s", userInfo.Name, arg)
				return
			}
		}
		fmt.Fprint(w, "command not found")
		return
	case "access-level":
		lvlStr := r.FormValue("level")
		lvl, err := strconv.Atoi(lvlStr)
		if err != nil {
			fmt.Fprint(w, "invalid level")
			return
		}
		cfg.Local.Options.MembersOnly = false
		cfg.Local.Options.RegularsOnly = false
		if lvl >= 2 {
			cfg.Local.Options.RegularsOnly = true
		} else if lvl == 1 {
			cfg.Local.Options.MembersOnly = true
		}
		cfg.WriteLCfg()
	case "player-level":
		name := strings.ToLower(r.FormValue("name"))
		if name == "" {
			fmt.Fprint(w, "name required")
			return
		}
		lvlStr := r.FormValue("level")
		lvl, err := strconv.Atoi(lvlStr)
		if err != nil {
			fmt.Fprint(w, "invalid level")
			return
		}
		reason := r.FormValue("reason")
		if reason == "" {
			reason = "No reason given"
		}
		old := fact.PlayerLevelGet(name, false)
		glob.PlayerListLock.RLock()
		pl := glob.PlayerList[name]
		glob.PlayerListLock.RUnlock()
		if pl == nil {
			fmt.Fprint(w, "player not found")
			return
		}
		if lvl >= 0 && old == -1 {
			fact.WriteUnban(name)
		}
		if lvl == -1 && old != -1 {
			reasonString := fmt.Sprintf("%v -- Panel", reason)
			fact.WriteBan(name, reasonString)
			pl.BanReason = reasonString
		}
		fact.AutoPromote(name, false, false)
		fact.PlayerLevelSet(pl.Name, lvl, true)
		fact.SetPlayerListDirty()
		fmt.Fprintf(w, "level set to %v", lvl)
		cwlog.DoLogAudit("%v: player-level %s %d", userInfo.Name, pl.Name, lvl)
		return
	default:
		fmt.Fprintf(w, "Command '%s' not supported via panel", cmd)
		return
	}

	fmt.Fprintf(w, "Command '%s' executed", cmd)
	cwlog.DoLogAudit("%v: ran %s", userInfo.Name, cmd)

	userInfo.Time = time.Now().Unix()
}

func buildInfoString() string {
	var lines []string
	skip := map[string]struct{}{
		constants.ProgName + " version": {},
		"ChatWire up-time":              {},
		"Next map reset":                {},
		"Time till reset":               {},
		"Interval":                      {},
		"UPS Average":                   {},
		"Connect via Steam":             {},
	}
	add := func(k, v string) {
		if v == "" || v == "0" || v == constants.Unknown || v == "(not configured)" {
			return
		}
		if _, ok := skip[k]; ok {
			return
		}
		lines = append(lines, fmt.Sprintf("%s: %s", k, v))
	}

	add(constants.ProgName+" version", constants.Version)
	add("SoftMod version", glob.SoftModVersion)
	add("Factorio version", fact.FactorioVersion)

	now := time.Now().Round(time.Second)
	add("ChatWire up-time", now.Sub(glob.Uptime.Round(time.Second)).Round(time.Second).String())
	if !fact.FactorioBootedAt.IsZero() && fact.FactorioBooted {
		add("Factorio up-time", now.Sub(fact.FactorioBootedAt.Round(time.Second)).Round(time.Second).String())
	}

	if cfg.Local.Options.PlayHourEnable {
		add("Time restrictions", fmt.Sprintf("%d - %d GMT", cfg.Local.Options.PlayStartHour, cfg.Local.Options.PlayEndHour))
	}
	add("Save name", fact.LastSaveName)
	add("Map time", fact.GametimeString)
	if fact.NumPlayers > 0 {
		add("Players online", fmt.Sprintf("%d", fact.NumPlayers))
	}

	if fact.HasResetTime() {
		add("Next map reset", cfg.Local.Options.NextReset.Local().Format("Jan 02 2006 03:04PM"))
		add("Time till reset", fact.TimeTillReset())
		if fact.HasResetInterval() {
			add("Interval", fact.FormatResetInterval())
		}
	}

	ten, thirty, hour := fact.GetFactUPS()
	if hour > 0 {
		add("UPS Average", fmt.Sprintf("10m: %.2f, 30m: %.2f, 1h: %.2f", ten, thirty, hour))
	} else if thirty > 0 {
		add("UPS Average", fmt.Sprintf("10m: %.2f, 30m: %.2f", ten, thirty))
	} else if ten > 0 {
		add("UPS Average", fmt.Sprintf("10m: %.2f", ten))
	} else {
		add("UPS Average", "no data")
	}

	glob.PlayerListLock.RLock()
	var mem, reg, vet, mod, ban int
	for _, player := range glob.PlayerList {
		switch player.Level {
		case -1:
			ban++
		case 1:
			mem++
		case 2:
			reg++
		case 3:
			vet++
		case 255:
			mod++
		}
	}
	bCount := len(banlist.BanList)
	ban += bCount
	glob.PlayerListLock.RUnlock()
	total := ban + mem + reg + vet + mod
	add("Members", fmt.Sprintf("%d", mem))
	add("Regulars", fmt.Sprintf("%d", reg))
	add("Veterans", fmt.Sprintf("%d", vet))
	add("Moderators", fmt.Sprintf("%d", mod))
	add("Banned", fmt.Sprintf("%d", ban))
	add("Total players", fmt.Sprintf("%d", total))

	if fact.PausedTicks > 4 {
		lines = append(lines, "Server is paused")
	}

	if url, ok := fact.MakeSteamURL(); ok {
		add("Connect via Steam", url)
	}

	return strings.Join(lines, "\n")
}

func cfgLines(v reflect.Value, prefix string, skip map[string]struct{}) []string {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	t := v.Type()
	var items []struct {
		name  string
		lines []string
	}
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" || f.Tag.Get("json") == "-" {
			continue
		}
		if tag := f.Tag.Get("form"); tag == "-" || tag == "hidden" {
			continue
		}
		name := f.Tag.Get("web")
		if name == "" {
			name = f.Name
		}
		if skip != nil {
			if _, ok := skip[name]; ok {
				continue
			}
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Struct {
			sub := cfgLines(fv, prefix+"  ", skip)
			items = append(items, struct {
				name  string
				lines []string
			}{name: name, lines: sub})
		} else {
			items = append(items, struct {
				name  string
				lines []string
			}{name: name, lines: []string{fmt.Sprintf("%v", fv.Interface())}})
		}
	}
	// Preserve struct order rather than sorting alphabetically so related
	// fields can be grouped together in the output.
	var out []string
	for _, it := range items {
		if len(it.lines) == 1 {
			out = append(out, fmt.Sprintf("%s%s: %s", prefix, it.name, strings.TrimSpace(it.lines[0])))
		} else {
			out = append(out, fmt.Sprintf("%s%s:", prefix, it.name))
			for _, l := range it.lines {
				out = append(out, prefix+"  "+l)
			}
		}
	}
	return out
}

func buildCfgStringSkip(i interface{}, skip map[string]struct{}) string {
	lines := cfgLines(reflect.ValueOf(i), "", skip)
	return strings.Join(lines, "\n")
}

func buildCfgString(i interface{}) string { return buildCfgStringSkip(i, nil) }

func setCfgField(path []string, value string) error {
	v := reflect.ValueOf(&cfg.Local).Elem()
	for i, p := range path {
		v = v.FieldByName(p)
		if !v.IsValid() {
			return fmt.Errorf("field not found")
		}
		if i == len(path)-1 {
			if !v.CanSet() {
				return fmt.Errorf("cannot set field")
			}
			switch v.Kind() {
			case reflect.String:
				v.SetString(value)
			case reflect.Bool:
				b := strings.ToLower(value)
				v.SetBool(b == "true" || b == "1" || b == "on" || b == "yes")
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return err
				}
				v.SetInt(n)
			case reflect.Float32, reflect.Float64:
				f, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return err
				}
				v.SetFloat(f)
			default:
				return fmt.Errorf("unsupported type")
			}
			return nil
		}
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
	}
	return fmt.Errorf("invalid path")
}

func fakeInteraction(u *glob.PanelTokenData) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:      discordgo.InteractionApplicationCommand,
			Member:    &discordgo.Member{User: &discordgo.User{ID: u.DiscID, Username: u.Name}},
			ChannelID: cfg.Local.Channel.ChatChannel,
			GuildID:   cfg.Global.Discord.Guild,
		},
	}
}

func generateCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	dnsName := cfg.Global.Paths.URLs.Domain
	if glob.LocalTestMode != nil && *glob.LocalTestMode {
		dnsName = "127.0.0.1"
	}
	tpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{Organization: []string{"ChatWire"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), DNSNames: []string{dnsName}, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPEM, keyPEM)
}
