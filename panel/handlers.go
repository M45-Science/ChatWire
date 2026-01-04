package panel

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/M45-Science/rcon"
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/commands"
	"ChatWire/commands/moderator"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
)

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
