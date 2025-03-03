package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

const modSettingsName = "mod-settings.dat"

func UploadFile(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	var msg *discordgo.Message
	msgDelay := time.Second * 3
	disc.InteractionEphemeralResponse(i, "Status", "Processing, please wait...")

	//Just in case
	factUpdater.FetchLock.Lock()
	defer factUpdater.FetchLock.Unlock()

	var modSettingsData []byte

	//Preprocessing
	var foundOption, foundSave, foundModList bool
	for _, item := range i.ApplicationCommandData().Options {
		tName := item.Name

		attachmentID := item.Value.(string)
		attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL

		switch tName {
		case "save-game":
			foundSave = true
		case "mod-list":
			foundModList = true
		case "mod-settings":
			msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "Your "+tName+" file is uploading.", glob.COLOR_GREEN)

			//We do this first, as we need it when we restart for the map.
			data, name, err := factUpdater.HttpGet(attachmentUrl)
			if err != nil {
				msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "**Your "+tName+" file failed while downloading.**", glob.COLOR_RED)
				cwlog.DoLogCW("Upload: Write http-get: Error: %v", err)
				time.Sleep(msgDelay)
				continue
			}
			if name == modSettingsName {
				modSettingsData = data
			} else {
				msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "**Your "+tName+" file didn't have the correct name: "+modSettingsName+"**", glob.COLOR_RED)
				time.Sleep(msgDelay)
			}
		default:
			continue
		}
	}

	//Processing
	for _, item := range i.ApplicationCommandData().Options {
		tName := item.Name

		attachmentID := item.Value.(string)
		attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL

		switch tName {
		case "save-game":
			foundOption = true
			msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "Your "+tName+" file is uploading.", glob.COLOR_GREEN)

			data, name, err := factUpdater.HttpGet(attachmentUrl)
			if err != nil {
				msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "**Your "+tName+" file failed while downloading.**", glob.COLOR_RED)
				cwlog.DoLogCW("Upload: http-get save-game: Error: %v", err)
				time.Sleep(msgDelay)
				continue
			}

			sBuf := fmt.Sprintf("Downloaded %v: %v, Size: %v", tName, name, humanize.Bytes(uint64(len(data))))
			msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", sBuf, glob.COLOR_GREEN)

			fileName := fmt.Sprintf("upload-%v-%v.zip", i.Member.User.ID, time.Now().UnixMilli())
			savePath := cfg.Global.Paths.Folders.ServersRoot +
				cfg.Global.Paths.ChatWirePrefix +
				cfg.Local.Callsign + "/" +
				cfg.Global.Paths.Folders.FactorioDir + "/" +
				cfg.Global.Paths.Folders.Saves + "/"
			filePath := savePath + fileName
			err = os.WriteFile(filePath, data, 0644)
			if err != nil {
				msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "**Your "+tName+" file failed while writing.**", glob.COLOR_RED)
				cwlog.DoLogCW("Upload: Write save-game: Error: %v", err)
				time.Sleep(msgDelay)
				continue
			}
			if len(modSettingsData) > 0 {
				modPath := cfg.Global.Paths.Folders.ServersRoot +
					cfg.Global.Paths.ChatWirePrefix +
					cfg.Local.Callsign + "/" +
					cfg.Global.Paths.Folders.FactorioDir + "/" +
					constants.ModsFolder + "/"
				err = os.WriteFile(modPath+modSettingsName, data, 0644)
				if err != nil {
					msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "**Your "+tName+" file failed while writing.**", glob.COLOR_RED)
					time.Sleep(msgDelay)
					cwlog.DoLogCW("Upload: Write mod-settings: Error: %v", err)
					continue
				}
				msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "Your "+modSettingsName+" has been loaded.", glob.COLOR_RED)
			}
			fact.DoChangeMap(strings.TrimSuffix(fileName, ".zip"))
		case "mod-list":
			if foundModList && foundSave {
				msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "You do not need to include a mod-list.json when uploading a save-game, ignoring.", glob.COLOR_GREEN)
				continue
			}
		case "mod-settings":
			continue
		default:
			continue
		}

	}
	if !foundOption {
		disc.InteractionEphemeralResponse(i, "Error:", "You must supply a file to upload.")
	}
}
