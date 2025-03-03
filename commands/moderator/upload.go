package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/support"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

const modSettingsName = "mod-settings.dat"

var UploadLock sync.Mutex

func UploadFile(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	UploadLock.Lock()
	defer UploadLock.Unlock()

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
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "Your "+tName+" file is uploading.", glob.COLOR_GREEN)

			//We do this first, as we need it when we restart for the map.
			data, name, err := factUpdater.HttpGet(attachmentUrl)
			if err != nil {
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "**Your "+tName+" file failed while downloading.**", glob.COLOR_RED)
				cwlog.DoLogCW("Upload: Write http-get: Error: %v", err)
				time.Sleep(msgDelay)
				continue
			}
			if name == modSettingsName {
				modSettingsData = data
			} else {
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "**Your "+tName+" file didn't have the correct name: "+modSettingsName+"**", glob.COLOR_RED)
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
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "Your "+tName+" file is uploading.", glob.COLOR_GREEN)

			data, name, err := factUpdater.HttpGet(attachmentUrl)
			if err != nil {
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "**Your "+tName+" file failed while downloading.**", glob.COLOR_RED)
				cwlog.DoLogCW("Upload: http-get save-game: Error: %v", err)
				time.Sleep(msgDelay)
				continue
			}

			sBuf := fmt.Sprintf("Downloaded %v: %v, Size: %v", tName, name, humanize.Bytes(uint64(len(data))))
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", sBuf, glob.COLOR_GREEN)

			if fact.FactorioBooted || fact.FactIsRunning {

				/* Turn off skip reset flag regardless of reset reason */
				if cfg.Local.Options.SkipReset {
					cfg.Local.Options.SkipReset = false
					cfg.WriteLCfg()
				}

				cfg.Local.Options.SkipReset = false
				fact.QueueReboot = false      //Skip queued reboot
				fact.QueueFactReboot = false  //Skip queued reboot
				fact.DoUpdateFactorio = false //Skip queued updates
				cfg.WriteLCfg()

				fact.SetAutolaunch(false, false)
				fact.QuitFactorio("Server rebooting for new custom map.")
				fact.WaitFactQuit(false)
			}

			saveFileName := fmt.Sprintf("upload-%v-%v.zip", i.Member.User.ID, time.Now().UnixMilli())
			savePath := cfg.Global.Paths.Folders.ServersRoot +
				cfg.Global.Paths.ChatWirePrefix +
				cfg.Local.Callsign + "/" +
				cfg.Global.Paths.Folders.FactorioDir + "/" +
				cfg.Global.Paths.Folders.Saves + "/"
			saveFilePath := savePath + saveFileName
			err = os.WriteFile(saveFilePath, data, 0644)
			if err != nil {
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "**Your "+tName+" file failed while writing.**", glob.COLOR_RED)
				cwlog.DoLogCW("Upload: Write save-game: Error: %v", err)
				time.Sleep(msgDelay)
				continue
			}
			//Touch save-game
			currentTime := time.Now().UTC().Local()
			_ = os.Chtimes(saveFilePath, currentTime, currentTime)

			if len(modSettingsData) > 0 {
				modPath := cfg.Global.Paths.Folders.ServersRoot +
					cfg.Global.Paths.ChatWirePrefix +
					cfg.Local.Callsign + "/" +
					cfg.Global.Paths.Folders.FactorioDir + "/" +
					constants.ModsFolder + "/"
				msPath := modPath + modSettingsName
				err = os.WriteFile(msPath, data, 0644)
				if err != nil {
					glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "**Your "+tName+" file failed while writing.**", glob.COLOR_RED)
					time.Sleep(msgDelay)
					cwlog.DoLogCW("Upload: Write mod-settings: Error: %v", err)
					continue
				}

				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "Your "+modSettingsName+" has been loaded.", glob.COLOR_RED)
			}
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "**Downloading any save-game installed mods, please wait...**", glob.COLOR_GREEN)

			if !support.SyncMods(saveFileName) {
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "mod-sync failed, attempting to continue.", glob.COLOR_RED)
				time.Sleep(msgDelay)
			}
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "Checking for mod updates.", glob.COLOR_GREEN)
			modupdate.CheckMods(true, true)
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "Attempting to load your save-game.", glob.COLOR_GREEN)
			fact.DoChangeMap(strings.TrimSuffix(saveFileName, ".zip"))
		case "mod-list":
			if foundModList && foundSave {
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "You do not need to include a mod-list.json when uploading a save-game, ignoring.", glob.COLOR_GREEN)
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
