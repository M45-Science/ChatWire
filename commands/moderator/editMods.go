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

	if fact.FactorioBooted || fact.FactIsRunning {
		buf := "Factorio is currently running. You must stop Factorio first."
		disc.InteractionEphemeralResponse(i, "Error:", buf)
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
		case "list":
			tmsg = tmsg + listMods(installedMods)
			//		case "add":
			//		case "remove":
		case "enable":
			installedMods, msg = ToggleMod(i, installedMods, option.StringValue(), true)
			tmsg = tmsg + msg + "\n"
		case "disable":
			installedMods, msg = ToggleMod(i, installedMods, option.StringValue(), false)
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
	if name == "" {
		emsg := "You must specify a mod(s) to " + enableStr(value, false) + "."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return installedMods, emsg
	}

	//Remove spaces
	input := strings.ReplaceAll(name, " ", "")
	parts := strings.Split(input, ",")

	emsg := ""
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
