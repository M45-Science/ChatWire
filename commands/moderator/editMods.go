package moderator

import (
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func EditMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

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
	for _, jmod := range jsonModList.Mods {
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

	//Handle commands
	for _, option := range i.ApplicationCommandData().Options {
		oName := strings.ToLower(option.Name)
		switch oName {
		case "add":
		case "remove":
			//
		case "enable":
			installedMods = ToggleMod(i, installedMods, oName, true)
		case "disable":
			installedMods = ToggleMod(i, installedMods, oName, false)
		}
	}

	//Write new mod list here

	//Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "The game has no installed mods."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return
	}
}

func ToggleMod(i *discordgo.InteractionCreate, installedMods []modupdate.ModData, name string, value bool) []modupdate.ModData {
	if name == "" {
		emsg := "You must specify a mod(s) to " + enableStr(value, false) + "."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return installedMods
	}
	for m, mod := range installedMods {
		//Remove spaces
		input := strings.ReplaceAll(name, " ", "")
		parts := strings.Split(input, ",")

		for _, part := range parts {
			if strings.EqualFold(mod.Name, part) {
				if mod.Enabled == value {
					emsg := "The mod '" + mod.Name + "' is now " + enableStr(value, true) + "."
					disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_GREEN)
					installedMods[m].Enabled = false
				} else {
					emsg := "The mod '" + mod.Name + "' was already " + enableStr(value, true) + "!"
					disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
				}
				break
			}
		}
	}

	return installedMods
}

func AddMod(name string, installedMods []modupdate.ModData) []modupdate.ModData {
	return installedMods
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
