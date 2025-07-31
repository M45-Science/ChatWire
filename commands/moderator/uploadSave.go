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
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

const msgTitle = "File Upload"

func handleCustomSave(i *discordgo.InteractionCreate, attachmentUrl string, modSettingsBytes []byte) {
	foundOption = true
	glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
		"The "+saveGameName+" file is uploading.", glob.COLOR_CYAN))

	saveGameBytes, name, err := factUpdater.HttpGet(true, attachmentUrl, true)
	if err != nil {
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
			"**The "+saveGameName+" file failed while downloading.**", glob.COLOR_RED))
		cwlog.DoLogCW("Upload: http-get "+saveGameName+": Error: %v", err)
		time.Sleep(constants.ErrMsgDelay)
		return
	}

	sBuf := fmt.Sprintf("Downloaded "+saveGameName+": %v, Size: %v", name, humanize.Bytes(uint64(len(saveGameBytes))))
	glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
		sBuf, glob.COLOR_CYAN))

	stopWaitFact("Server rebooting to load a new custom map.")

	saveFileName := fmt.Sprintf("upload-%v-%v.zip", i.Member.User.ID, time.Now().UnixMilli())
	if insertSaveGame(i, saveFileName, saveGameBytes) {
		return
	}

	insertModSettings(modSettingsBytes)

	glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
		"**Downloading any "+saveGameName+" installed mods, PLEASE WAIT...**", glob.COLOR_CYAN))
	if !support.SyncMods(i, saveFileName) {
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
			"mod-sync failed, attempting to continue.", glob.COLOR_RED))
		time.Sleep(constants.ErrMsgDelay)
	}
	glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
		"Checking for mod updates.", glob.COLOR_CYAN))
	modupdate.CheckMods(true, true)
	glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
		"Attempting to load the "+saveGameName+".", glob.COLOR_CYAN))
	fact.DoChangeMap(strings.TrimSuffix(saveFileName, ".zip"))
}

func insertSaveGame(i *discordgo.InteractionCreate, saveFileName string, saveGameData []byte) bool {
	savePath := cfg.GetSavesFolder() + "/"
	saveFilePath := savePath + saveFileName
	err := os.WriteFile(saveFilePath, saveGameData, 0644)
	if err != nil {
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
			"**The "+saveGameName+" file failed while writing.**", glob.COLOR_RED))
		cwlog.DoLogCW("Upload: Write "+saveGameName+": Error: %v", err)
		time.Sleep(constants.ErrMsgDelay)
		return true
	}
	//Touch save-game
	currentTime := time.Now().UTC().Local()
	_ = os.Chtimes(saveFilePath, currentTime, currentTime)

	glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle,
		"Checking "+saveGameName+".", glob.COLOR_CYAN))
	if fact.FileHasZipBomb(saveFilePath) {
		msg := "**THE " + strings.ToUpper(saveGameName) + " MAY CONTAIN A ZIP-BOMB ATTACK, ABORTING. UPLOADED BY: ID: " + i.Member.User.ID + " USERNAME: " + i.Member.User.Username + " INCIDENT LOGGED. **"
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), msgTitle, msg, glob.COLOR_RED))
		cwlog.DoLogAudit(msg)
		if err := os.Remove(saveFilePath); err != nil {
			cwlog.DoLogCW("uploadSave: failed to remove suspicious save: %v", err)
		}
		return true
	}
	return false
}
