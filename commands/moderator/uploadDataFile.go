package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"bytes"
	"encoding/binary"
	"os"
	"strconv"
	"time"
)

const maxModsAllowed = 150

func handleModList(modListBytes []byte) {
	if foundModList && foundSave {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**You do not need to include a "+constants.ModListName+" when uploading a "+saveGameName+", ignoring.**", glob.COLOR_ORANGE)
		time.Sleep(errMsgDelay)
		return
	}
	if len(modListBytes) > 0 {
		savePath := cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			cfg.Global.Paths.Folders.Mods + "/"
		modListPath := savePath + constants.ModListName

		err := os.WriteFile(modListPath, modListBytes, 0655)
		if err != nil {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Your "+constants.ModListName+" file failed while writing.**", glob.COLOR_RED)
			return
		}
		listMods, err := modupdate.GetModList()
		enabledCount := 0
		disabledCount := 0
		enabledModList := ""
		for _, item := range listMods.Mods {
			if !modupdate.IsBaseMod(item.Name) {
				if item.Enabled {
					enabledCount++
					if enabledModList != "" {
						enabledModList = enabledModList + ", "
					}
					enabledModList = enabledModList + item.Name
				} else {
					disabledCount++
				}
			}
		}
		totalCount := enabledCount + disabledCount
		if err != nil || totalCount == 0 {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Your "+constants.ModListName+" file contains invalid data or no mods!**", glob.COLOR_RED)
			return
		}
		if enabledCount > maxModsAllowed || disabledCount > maxModsAllowed {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Your "+constants.ModListName+" file contains too many mods! ("+strconv.FormatInt(maxModsAllowed, 10)+")**", glob.COLOR_RED)
			return
		}
		if enabledCount > 0 {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"Downloading: "+enabledModList, glob.COLOR_GREEN)
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Downloading the "+strconv.FormatInt(int64(enabledCount), 10)+" enabled mods in your "+constants.ModListName+" file, PLEASE WAIT...**", glob.COLOR_GREEN)
			modupdate.CheckMods(true, true)
		} else {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Your "+constants.ModListName+" file contains no enabled mods!**", glob.COLOR_RED)
			return
		}
	}
}

func handleDataFile(attachmentUrl, typeName string) []byte {
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
		"Your "+typeName+" file is uploading.", glob.COLOR_GREEN)

	//We do this first, as we need it when we restart for the map.
	data, name, err := factUpdater.HttpGet(true, attachmentUrl, false)
	if err != nil {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your "+typeName+" file failed while downloading.**", glob.COLOR_RED)
		cwlog.DoLogCW("Upload: "+typeName+": http-get: Error: %v", err)
		time.Sleep(errMsgDelay)
		return nil
	}
	if name == typeName {
		if len(data) > MaxModSettingsSize {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**The "+typeName+" is too large, skipping... **", glob.COLOR_RED)
			return nil
		}
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"Downloaded "+typeName+".", glob.COLOR_GREEN)
		return data
	} else {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"**Your "+typeName+" file didn't have the correct name.**", glob.COLOR_RED)
		time.Sleep(errMsgDelay)
	}
	return nil
}

func insertModSettings(modSettingsData []byte) bool {
	if len(modSettingsData) > 0 {
		if verifyModSettings(modSettingsData) {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Your "+constants.ModSettingsName+" contains invalid data, ABORTING.**", glob.COLOR_RED)
			return true
		}

		modPath := cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			constants.ModsFolder + "/"
		msPath := modPath + constants.ModSettingsName
		err := os.WriteFile(msPath, modSettingsData, 0644)
		if err != nil {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
				"**Your "+constants.ModSettingsName+" file failed while writing.**", glob.COLOR_RED)
			time.Sleep(errMsgDelay)
			cwlog.DoLogCW("Upload: Write "+constants.ModSettingsName+": Error: %v", err)
			return true
		}

		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status",
			"Your "+constants.ModSettingsName+" has been loaded.", glob.COLOR_GREEN)
	}
	return false
}

func verifyModSettings(data []byte) bool {

	var major, minor, patch, dev uint16
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.LittleEndian, &major)
	binary.Read(reader, binary.LittleEndian, &minor)
	binary.Read(reader, binary.LittleEndian, &patch)
	binary.Read(reader, binary.LittleEndian, &dev)

	if dev != 0 || (major == 0 && minor < 12) {
		cwlog.DoLogCW("verifyModSettings: Invalid header.")
		return true
	}

	return false
}
