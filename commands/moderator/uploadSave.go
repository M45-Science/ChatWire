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
	"ChatWire/util"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

func handleCustomSave(i *discordgo.InteractionCreate, attachmentUrl string, modSettingsBytes []byte) {
	foundOption = true
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Your "+saveGameName+" file is uploading.", glob.COLOR_GREEN)

	saveGameBytes, name, err := factUpdater.HttpGet(true, attachmentUrl, true)
	if err != nil {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your "+saveGameName+" file failed while downloading.**", glob.COLOR_RED)
		cwlog.DoLogCW("Upload: http-get "+saveGameName+": Error: %v", err)
		time.Sleep(constants.ErrMsgDelay)
		return
	}

	sBuf := fmt.Sprintf("Downloaded "+saveGameName+": %v, Size: %v", name, humanize.Bytes(uint64(len(saveGameBytes))))
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		sBuf, glob.COLOR_GREEN)

	stopWaitFact("Server rebooting to load a new custom map.")

	saveFileName := fmt.Sprintf("upload-%v-%v.zip", i.Member.User.ID, time.Now().UnixMilli())
	if insertSaveGame(i, saveFileName, saveGameBytes) {
		return
	}

	insertModSettings(modSettingsBytes)

	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"**Downloading any "+saveGameName+" installed mods, PLEASE WAIT...**", glob.COLOR_GREEN)
	if !support.SyncMods(saveFileName) {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"mod-sync failed, attempting to continue.", glob.COLOR_RED)
		time.Sleep(constants.ErrMsgDelay)
	}
	modBuf := showSyncMods()
	if modBuf != "" {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"Installed mods: "+modBuf, glob.COLOR_GREEN)
	}
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Checking for mod updates.", glob.COLOR_GREEN)
	modupdate.CheckMods(true, true)
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Attempting to load your "+saveGameName+".", glob.COLOR_GREEN)
	fact.DoChangeMap(strings.TrimSuffix(saveFileName, ".zip"))
}

func insertSaveGame(i *discordgo.InteractionCreate, saveFileName string, saveGameData []byte) bool {
	savePath := util.GetSavesFolder() + "/"
	saveFilePath := savePath + saveFileName
	err := os.WriteFile(saveFilePath, saveGameData, 0644)
	if err != nil {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your "+saveGameName+" file failed while writing.**", glob.COLOR_RED)
		cwlog.DoLogCW("Upload: Write "+saveGameName+": Error: %v", err)
		time.Sleep(constants.ErrMsgDelay)
		return true
	}
	//Touch save-game
	currentTime := time.Now().UTC().Local()
	_ = os.Chtimes(saveFilePath, currentTime, currentTime)

	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Checking "+saveGameName+".", glob.COLOR_GREEN)
	if fact.FileHasZipBomb(saveFilePath) {
		msg := "**THE " + strings.ToUpper(saveGameName) + " MAY CONTAIN A ZIP-BOMB ATTACK, ABORTING. UPLOADED BY: ID: " + i.Member.User.ID + " USERNAME: " + i.Member.User.Username + " INCIDENT LOGGED. **"
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", msg, glob.COLOR_RED)
		cwlog.DoLogCW(msg)
		cwlog.DoLogGame(msg)
		os.Remove(saveFilePath)
		return true
	}
	return false
}
