package panel

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

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

	paused := fact.PausedTicks > 4

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
	bCount := banlist.Count()
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
