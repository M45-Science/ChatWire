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
const MaxModSettingsSize = 1024 * 1024 //1MB
const MaxModListSize = 1024 * 1024     //1MB

var (
	UploadLock                           sync.Mutex
	foundOption, foundSave, foundModList bool
	errMsgDelay                          = time.Second * 3
)

func UploadFile(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	//Just in case
	UploadLock.Lock()
	defer UploadLock.Unlock()

	disc.InteractionEphemeralResponse(i, "Status",
		"Processing, please wait...")

	//Just in case
	factUpdater.FetchLock.Lock()
	defer factUpdater.FetchLock.Unlock()

	var modSettingsData []byte

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
		case "mod-settings":
			modSettingsData = handleModSettings(attachmentUrl)
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
			handleCustomSave(i, attachmentUrl, modSettingsData)
		case "mod-list":
			if foundModList && foundSave {
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
					"**You do not need to include a mod-list.json when uploading a save-game, ignoring.**", glob.COLOR_ORANGE)
				time.Sleep(errMsgDelay)
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

func handleCustomSave(i *discordgo.InteractionCreate, attachmentUrl string, modSettingsData []byte) {
	foundOption = true
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Your save-game file is uploading.", glob.COLOR_GREEN)

	data, name, err := factUpdater.HttpGet(attachmentUrl)
	if err != nil {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your save-game file failed while downloading.**", glob.COLOR_RED)
		cwlog.DoLogCW("Upload: http-get save-game: Error: %v", err)
		time.Sleep(errMsgDelay)
		return
	}

	sBuf := fmt.Sprintf("Downloaded save-game: %v, Size: %v", name, humanize.Bytes(uint64(len(data))))
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		sBuf, glob.COLOR_GREEN)

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
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your save-game file failed while writing.**", glob.COLOR_RED)
		cwlog.DoLogCW("Upload: Write save-game: Error: %v", err)
		time.Sleep(errMsgDelay)
		return
	}
	//Touch save-game
	currentTime := time.Now().UTC().Local()
	_ = os.Chtimes(saveFilePath, currentTime, currentTime)

	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Checking save-game.", glob.COLOR_GREEN)
	if fact.HasZipBomb(saveFilePath) {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**THE SAVE-GAME MAY CONTAIN A ZIP-BOMB ATTACK, ABORTING.**", glob.COLOR_RED)
		os.Remove(saveFilePath)
		return
	}

	if len(modSettingsData) > 0 {
		modPath := cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			constants.ModsFolder + "/"
		msPath := modPath + modSettingsName
		err = os.WriteFile(msPath, modSettingsData, 0644)
		if err != nil {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Your "+modSettingsName+" file failed while writing.**", glob.COLOR_RED)
			time.Sleep(errMsgDelay)
			cwlog.DoLogCW("Upload: Write mod-settings: Error: %v", err)
			return
		}

		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"Your "+modSettingsName+" has been loaded.", glob.COLOR_RED)
	}
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"**Downloading any save-game installed mods, PLEASE WAIT...**", glob.COLOR_GREEN)
	modBuf := showSyncMods()
	if modBuf != "" {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"Installed mods: "+modBuf, glob.COLOR_GREEN)
	}
	if !support.SyncMods(saveFileName) {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"mod-sync failed, attempting to continue.", glob.COLOR_RED)
		time.Sleep(errMsgDelay)
	}
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Checking for mod updates.", glob.COLOR_GREEN)
	modupdate.CheckMods(true, true)
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Attempting to load your save-game.", glob.COLOR_GREEN)
	fact.DoChangeMap(strings.TrimSuffix(saveFileName, ".zip"))
}

func showSyncMods() string {
	buf := ""
	modList, mErr := support.GetGameMods()
	if mErr == nil && modList != nil {
		for _, mod := range modList.Mods {
			if strings.EqualFold(mod.Name, "base") {
				continue
			}
			if !mod.Enabled {
				continue
			}
			if buf != "" {
				buf = buf + ", "
			}
			if mod.Enabled {
				buf = buf + strings.TrimSuffix(mod.Name, ".zip")
			}
		}
	}

	if buf == "" {
		buf = strings.Join(support.GetModFiles(), ", ")
	}

	return buf
}

func handleModSettings(attachmentUrl string) []byte {
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Your "+modSettingsName+" file is uploading.", glob.COLOR_GREEN)

	//We do this first, as we need it when we restart for the map.
	data, name, err := factUpdater.HttpGet(attachmentUrl)
	if err != nil {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your "+modSettingsName+" file failed while downloading.**", glob.COLOR_RED)
		cwlog.DoLogCW("Upload: Write http-get: Error: %v", err)
		time.Sleep(errMsgDelay)
		return nil
	}
	if name == modSettingsName {
		if len(data) > MaxModSettingsSize {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**The "+modSettingsName+" is too large, skipping... **", glob.COLOR_RED)
			return nil
		}
		return data
	} else {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your "+modSettingsName+" file didn't have the correct name.**", glob.COLOR_RED)
		time.Sleep(errMsgDelay)
	}
	return nil
}
