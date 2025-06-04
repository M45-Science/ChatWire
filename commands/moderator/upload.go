package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

func UploadFile(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	//Just in case
	uploadLock.Lock()
	defer uploadLock.Unlock()

	//Save current auto-mod-update setting, disable mod updating, then restore on exit.
	RestoreSetting := cfg.Local.Options.ModUpdate
	cfg.Local.Options.ModUpdate = false
	defer func(rVal bool) {
		cfg.Local.Options.ModUpdate = rVal
	}(RestoreSetting)

	disc.InteractionEphemeralResponse(i, "Status",
		"Processing, please wait...")

	glob.UpdatersLock.Lock()
	defer glob.UpdatersLock.Unlock()

	var modSettingsBytes, modListBytes []byte

	//Preprocessing
	for _, item := range i.ApplicationCommandData().Options {
		tName := item.Name

		attachmentID := item.Value.(string)
		attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL

		switch tName {
		case "save-game":
			foundSave = true
		case "mod-list":
			foundModList = true
			modListBytes = handleDataFile(attachmentUrl, constants.ModListName)
		case "mod-settings":
			modSettingsBytes = handleDataFile(attachmentUrl, constants.ModSettingsName)
		default:
			continue
		}
	}

	//Processing
	for _, item := range i.ApplicationCommandData().Options {
		tName := item.Name

		attachmentID := item.Value.(string)
		attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL

		var doLaunch bool

		switch tName {
		case "save-game":
			handleCustomSave(i, attachmentUrl, modSettingsBytes)
		case "mod-list":
			if !foundSave {
				stopWaitFact("Server rebooting to load a new a new " + constants.ModListName + " file.")
				handleModList(modListBytes)

				doLaunch = true
			}
		case "mod-settings":
			if !foundSave {
				stopWaitFact("Server rebooting to load new " + constants.ModSettingsName + " file.")
				insertModSettings(modSettingsBytes)

				doLaunch = true
			}
			continue
		default:
			continue
		}

		if doLaunch {
			glob.BootMessage = nil
			glob.RelaunchThrottle = 0
			fact.SetAutolaunch(true, false)
		}

	}
	if !foundOption {
		disc.InteractionEphemeralResponse(i, "Error:", "You must supply a file to upload.")
	}
}
