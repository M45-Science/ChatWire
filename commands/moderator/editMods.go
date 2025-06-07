package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modedit"
	"ChatWire/modupdate"
	"ChatWire/util"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func EditMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	var msg string
	tmsg := ""

	optionsList := i.ApplicationCommandData().Options
	if len(optionsList) == 0 {
		emsg := "You must choose at least one option."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return
	}

	for _, option := range optionsList {
		oName := strings.ToLower(option.Name)

		switch oName {
		case "mod-history":
			tmsg = tmsg + modupdate.ListHistory(false)
		case "full-history":
			tmsg = tmsg + modupdate.ListHistory(true)
		case "clear-history":
			tmsg = tmsg + modupdate.ClearHistory()
		case "show-mods":
			tmsg = tmsg + showMods()
		case "enable-mod":
			msg = toggleMod(i, option.StringValue(), true)
			tmsg = tmsg + msg + "\n"
		case "disable-mod":
			msg = toggleMod(i, option.StringValue(), false)
			tmsg = tmsg + msg + "\n"
		case "add-mod":
			tmsg = tmsg + addMod(i, option.StringValue())
		case "set-version-pref":
			tmsg = tmsg + setVersion(i, option.StringValue())
		case "clear-all-mods":
			msg = clearAllMods()
			tmsg = tmsg + msg + "\n"
		default:
			msg = oName + " is not a valid option."
			tmsg = tmsg + msg + "\n"
		}
	}

	//Check if we need to proceed
	if tmsg == "" {
		tmsg = "Unknown error"
	}

	if len(tmsg) > constants.MaxDiscordMsgLen {
		disc.InteractionEphemeralFileResponse(i, "Edit-Mods", "See attached history.", "mod-history.txt", []byte(tmsg))
	} else {
		disc.InteractionEphemeralResponseColor(i, "Edit-Mods", tmsg, glob.COLOR_CYAN)
	}
}

func clearAllMods() string {
	if fact.FactorioBooted || fact.FactIsRunning {
		emsg := "Factorio is currently running. You must stop Factorio first."
		return emsg
	}
	err := os.RemoveAll(cfg.GetModsFolder())
	if err != nil {
		return "Unable to delete mods folder: " + err.Error()
	}
	err = os.Mkdir(cfg.GetModsFolder(), 0755)
	if err != nil {
		return "Unable to create a new mods folder: " + err.Error()
	}

	modupdate.WriteModsList(modupdate.ModListData{})
	return "All mods, settings and old mods were deleted."
}

func addMod(i *discordgo.InteractionCreate, input string) string {

	input = strings.ReplaceAll(input, " ", "")
	mods := strings.Split(input, ",")

	buf := ""
	modList, _ := modupdate.GetModList()

	for _, mod := range mods {
		modName, success := parseModName(mod)
		if !success {
			return modName + ": mod not found."
		}
		for _, item := range modList.Mods {
			if item.Name == modName {
				return "The mod " + modName + " is already installed!"
			}
		}
		disc.InteractionEphemeralResponseColor(i, "Status", "Adding mods: "+modName, glob.COLOR_CYAN)

		modupdate.AddModHistory(modupdate.ModHistoryItem{
			Name: mod, Notes: "Added by " + i.Member.User.Username, Date: time.Now()})

		modList.Mods = append(modList.Mods, modupdate.ModData{Name: modName, Enabled: true})
		modupdate.WriteModsList(modList)

		if buf != "" {
			buf = buf + ", "
		}
		buf = buf + modName
	}

	if buf != "" {
		modupdate.CheckModUpdates(false)
		return "Adding mods: " + buf
	}

	return "You must specify mods to add."
}

func parseModName(input string) (string, bool) {
	name := strings.TrimSpace(input)

	if util.ContainsIgnoreCase(name, "factorio.com") {
		temp := util.TrimPrefixIgnoreCase(name, "https")
		temp = util.TrimPrefixIgnoreCase(temp, "http")
		temp = util.TrimPrefixIgnoreCase(temp, "://mods.factorio.com/mod/")
		temp = util.TrimPrefixIgnoreCase(temp, "://mods.factorio.com/download/")
		var parts []string
		if strings.Contains(temp, "?") {
			parts = strings.Split(temp, "?")
		} else {
			parts = strings.Split(temp, "/")
		}
		modName := parts[0]

		modData, err := modupdate.DownloadModInfo(modName)
		if err != nil {
			if strings.HasPrefix(err.Error(), "Mod info unmarshal failed") {
				return "Mod not found: " + modName, false
			} else {
				return "Error looking up mod: " + modName, false
			}
		}

		return modData.Name, true

	} else {
		modData, err := modupdate.DownloadModInfo(name)
		if err != nil {
			if strings.HasPrefix(err.Error(), "Mod info unmarshal failed") {
				return "Mod not found: " + name, false
			} else {
				return "Error looking up mod: " + name, false
			}
		}
		return modData.Name, true
	}
}

func showMods() string {
	modFiles, err := modupdate.GetModFiles()
	if err != nil {
		return "Unable to read mod files."
	}
	modList, _ := modupdate.GetModList()
	mergedMods := modupdate.MergeModLists(modFiles, modList)

	prefs := modedit.ReadPrefs()
	prefMap := map[string]string{}
	for _, p := range prefs.Mods {
		prefMap[strings.ToLower(p.Name)] = p.Version
	}

	ebuf := ""
	for _, item := range mergedMods {
		if item.Name == "base" || !item.Enabled {
			continue
		}
		if ebuf != "" {
			ebuf += "\n"
		}
		line := ""
		if modupdate.IsBaseMod(item.Name) {
			line = item.Name + " (base mod)"
		} else if item.Version != "" {
			ebuf = ebuf + item.Name + " (" + item.Version + ")"
		} else {
			line = item.Name
		}
		if v, ok := prefMap[strings.ToLower(item.Name)]; ok && v != "" {
			line += " **(FORCE " + v + ")**"
		}
		ebuf += line
	}

	dbuf := ""
	for _, item := range mergedMods {
		if item.Name == "base" || item.Enabled {
			continue
		}
		if dbuf != "" {
			dbuf += "\n"
		}
		line := ""
		if modupdate.IsBaseMod(item.Name) {
			line = item.Name + " (base mod)"
		} else if item.Version != constants.Unknown {
			line = item.Name + " (" + item.Version + ")"
		} else {
			line = item.Name
		}
		if v, ok := prefMap[strings.ToLower(item.Name)]; ok && v != "" {
			line += " (force " + v + ")"
		}
		dbuf += line
	}

	if ebuf == "" {
		ebuf = "Enabled: None"
	} else {
		ebuf = "Enabled:\n" + ebuf
	}

	if dbuf == "" {
		dbuf = "\n\nDisabled: None"
	} else {
		dbuf = "Disabled:\n" + dbuf
		if ebuf != "" {
			dbuf = "\n\n" + dbuf
		}
	}
	return ebuf + dbuf
}

func toggleMod(i *discordgo.InteractionCreate, name string, value bool) string {
	if fact.FactorioBooted || fact.FactIsRunning {
		emsg := "Factorio is currently running. You must stop Factorio first."
		return emsg
	}
	if name == "" {
		emsg := "You must specify a mod(s) to " + enableStr(value, false) + "."
		return emsg
	}

	modFiles, err := modupdate.GetModFiles()
	if err != nil {
		return "Unable to read mod files."
	}
	modList, _ := modupdate.GetModList()
	mergedMods := modupdate.MergeModLists(modFiles, modList)

	//Remove spaces
	input := strings.ReplaceAll(name, " ", "")
	parts := strings.Split(input, ",")

	emsg := ""
	dirty := false
	for _, part := range parts {
		found := false
		for m, mod := range mergedMods {
			if mod.Name == "base" {
				continue
			}
			if strings.EqualFold(mod.Name, part) {
				if mod.Enabled != value {
					emsg = emsg + "The mod '" + mod.Name + "' is now " + enableStr(value, true) + "."
					mergedMods[m].Enabled = value

					modupdate.AddModHistory(modupdate.ModHistoryItem{
						Name: mod.Name, Notes: enableStr(value, true) + " by " + i.Member.User.Username, Date: time.Now()})

					dirty = true

				} else {
					emsg = emsg + "The mod '" + mod.Name + "' was already " + enableStr(value, true) + "!"
				}
				found = true
				break
			}
		}
		if !found {
			emsg = emsg + "There is no mod by the name: " + input
		}
	}
	if dirty {
		modupdate.WriteModHistory()
		outMods := []modupdate.ModData{}
		for _, item := range mergedMods {
			outMods = append(outMods, modupdate.ModData{Name: item.Name, Enabled: item.Enabled})
		}
		outputList := modupdate.ModListData{Mods: outMods}
		modupdate.WriteModsList(outputList)
	}
	return emsg
}

/* Bool to string */
func enableStr(b bool, pastTense bool) string {
	if pastTense {
		if b {
			return "enabled"
		} else {
			return "disabled"
		}
	} else {
		if b {
			return "enable"
		} else {
			return "disable"
		}
	}
}

func setVersion(i *discordgo.InteractionCreate, input string) string {
	if input == "" {
		return "You must specify a mod=version pair."
	}
	input = strings.ReplaceAll(input, " ", "")
	parts := strings.Split(input, ",")

	modFiles, _ := modupdate.GetModFiles()
	valid := map[string]bool{}
	for _, m := range modFiles {
		valid[strings.ToLower(m.Name)] = true
	}

	buf := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		seg := strings.SplitN(p, "=", 2)
		if len(seg) != 2 {
			if buf != "" {
				buf += "\n"
			}
			buf += p + ": invalid format"
			continue
		}
		name := strings.ToLower(seg[0])
		vers := seg[1]
		if !valid[name] {
			if buf != "" {
				buf += "\n"
			}
			buf += seg[0] + ": mod not installed"
			continue
		}
		if err := modedit.SetVersion(name, vers); err != nil {
			if buf != "" {
				buf += "\n"
			}
			buf += seg[0] + ": " + err.Error()
			continue
		}
		if buf != "" {
			buf += "\n"
		}
		buf += seg[0] + " set to " + vers
	}

	modupdate.CheckModUpdates(false)
	return buf
}
