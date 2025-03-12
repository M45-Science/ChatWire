package moderator

import (
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/util"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func EditMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	var msg string

	//Read all mod.zip files
	modFileList, err := modupdate.GetModFiles()
	if err != nil {
		emsg := "Unable to read mods directory."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return
	}
	//Read mods-list.json
	jsonModList, err := modupdate.GetModList()
	if err != nil {
		emsg := "Unable to read the " + constants.ModListName + " file."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return
	}

	//Merge lists
	installedMods := jsonModList.Mods
	for _, jmod := range modFileList {
		found := false
		for _, fmod := range modFileList {
			if strings.EqualFold(jmod.Name, fmod.Name) {
				found = true
				break
			}
		}
		if !found {
			//Default enabled
			newMod := modupdate.ModData{Name: jmod.Name, Enabled: true}
			installedMods = append(installedMods, newMod)
		}
	}

	optionsList := i.ApplicationCommandData().Options
	if len(optionsList) == 0 {
		emsg := "You must choose at least one option."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return
	}
	tmsg := ""

	for _, option := range optionsList {
		oName := strings.ToLower(option.Name)

		switch oName {
		case "mod-history":
			tmsg = tmsg + modupdate.ListHistory()
		case "clear-history":
			tmsg = tmsg + modupdate.ClearHistory()
		case "list-mods":
			tmsg = tmsg + listMods(installedMods)
		case "enable-mod":
			installedMods, msg = ToggleMod(i, installedMods, option.StringValue(), true)
			tmsg = tmsg + msg + "\n"
		case "disable-mod":
			installedMods, msg = ToggleMod(i, installedMods, option.StringValue(), false)
			tmsg = tmsg + msg + "\n"
		case "add-mod":
			tmsg = tmsg + addMod(option.StringValue())
		case "clear-all-mods":
			msg = clearAllMods()
			tmsg = tmsg + msg + "\n"
		case "updater-blacklist":
			msg = clearAllMods()
			tmsg = tmsg + msg + "\n"
		default:
			msg = oName + " is not a valid option."
			tmsg = tmsg + msg + "\n"
		}
	}

	if !writeModsList(modupdate.ModListData{Mods: installedMods}) {
		emsg := "Failed to write the mod-list.json file."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return
	}

	//Check if we need to proceed
	if tmsg == "" {
		tmsg = "Unknown error"
	}

	disc.InteractionEphemeralResponseColor(i, "Status", tmsg, glob.COLOR_CYAN)
}

func updaterBlacklist(input string) string {
	return ""
}

func clearAllMods() string {
	if fact.FactorioBooted || fact.FactIsRunning {
		emsg := "Factorio is currently running. You must stop Factorio first."
		return emsg
	}
	//Clear all zips, json and dat.
	return ""
}

func addMod(input string) string {

	input = strings.ReplaceAll(input, " ", "")
	mods := strings.Split(input, ",")

	buf := ""
	modList, err := modupdate.GetModList()
	if err != nil {
		return err.Error()
	}

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
		modupdate.ModHistory = append(modupdate.ModHistory, modupdate.ModHistoryData{
			Name: mod, Notes: "Added by user", Date: time.Now()})
		modList.Mods = append(modList.Mods, modupdate.ModData{Name: modName, Enabled: true})
		if buf != "" {
			buf = buf + ", "
		}
		buf = buf + modName
	}

	if buf != "" {
		modupdate.WriteModHistory()
		writeModsList(modList)
		modupdate.CheckModUpdates(false)
		return "Adding mods: " + buf
	}

	return "You must specify mods to add."
}

func parseModName(input string) (string, bool) {
	name := strings.TrimSpace(input)

	if ContainsIgnoreCase(name, "factorio.com") {
		temp := TrimPrefixIgnoreCase(name, "https")
		temp = TrimPrefixIgnoreCase(temp, "http")
		temp = TrimPrefixIgnoreCase(temp, "://mods.factorio.com/mod/")
		temp = TrimPrefixIgnoreCase(temp, "://mods.factorio.com/download/")
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

func listMods(installedMods []modupdate.ModData) string {
	ebuf := ""
	for _, item := range installedMods {
		if strings.EqualFold(item.Name, "base") {
			continue
		}
		if !item.Enabled {
			continue
		}
		if ebuf != "" {
			ebuf = ebuf + ", "
		}
		ebuf = ebuf + item.Name
	}

	dbuf := ""
	for _, item := range installedMods {
		if strings.EqualFold(item.Name, "base") {
			continue
		}
		if item.Enabled {
			continue
		}
		if dbuf != "" {
			dbuf = dbuf + ", "
		}
		dbuf = dbuf + item.Name
	}

	if ebuf == "" {
		ebuf = "Enabled: None"
	} else {
		ebuf = "Enabled: " + ebuf
	}

	if dbuf == "" {
		dbuf = "\n\nDisabled: None"
	} else {
		dbuf = "Disabled: " + dbuf
		if ebuf != "" {
			dbuf = "\n\n" + dbuf
		}
	}
	return ebuf + dbuf
}

func ToggleMod(i *discordgo.InteractionCreate, installedMods []modupdate.ModData, name string, value bool) ([]modupdate.ModData, string) {
	if fact.FactorioBooted || fact.FactIsRunning {
		emsg := "Factorio is currently running. You must stop Factorio first."
		return installedMods, emsg
	}
	if name == "" {
		emsg := "You must specify a mod(s) to " + enableStr(value, false) + "."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return installedMods, emsg
	}

	//Remove spaces
	input := strings.ReplaceAll(name, " ", "")
	parts := strings.Split(input, ",")

	emsg := ""
	dirty := false
	for _, part := range parts {
		found := false
		for m, mod := range installedMods {
			if strings.EqualFold(mod.Name, "base") {
				continue
			}
			if strings.EqualFold(mod.Name, part) {
				if mod.Enabled != value {
					emsg = emsg + "The mod '" + mod.Name + "' is now " + enableStr(value, true) + "."
					installedMods[m].Enabled = value

					modupdate.ModHistory = append(modupdate.ModHistory, modupdate.ModHistoryData{
						Name: mod.Name, Notes: enableStr(value, true) + " by user", Date: time.Now()})
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
	}
	return installedMods, emsg
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

func writeModsList(modList modupdate.ModListData) bool {

	finalPath := util.GetModsFolder() + constants.ModListName
	tempPath := finalPath + ".tmp"

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(modList); err != nil {
		cwlog.DoLogCW("writeModsList: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("writeModsList: os.Create failure")
		return false
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("writeModsList: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("writeModsList: Couldn't rename " + constants.ModListName + ".tmp file.")
		return false
	}

	cwlog.DoLogCW("Wrote " + constants.ModListName)

	return true
}

func TrimPrefixIgnoreCase(s, prefix string) string {
	if strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
		return s[len(prefix):]
	}
	return s
}

func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(
		strings.ToLower(s), strings.ToLower(substr),
	)
}
