package factUpdater

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func DoQuickLatest(force bool) (*InfoData, string, bool, bool) {
	glob.UpdatersLock.Lock()
	defer glob.UpdatersLock.Unlock()

	glob.ResetUpdateMessage()
	opToken := fact.BeginOperation("Updating Factorio", "Checking for Factorio updates.")

	info := &InfoData{Xreleases: cfg.Local.Options.ExpUpdates, Build: "headless", Distro: "linux64"}

	if force {
		newVersion, err := quickLatest(info)
		if err != nil {
			fact.FailOperation(opToken, "Updating Factorio", "Checking latest Factorio release failed: "+err.Error(), glob.COLOR_RED)
			return info, fmt.Sprintf("quickLatest: %v", err.Error()), true, false
		}
		info.VersInt = *newVersion
		fact.UpdateOperation(opToken, "Updating Factorio", "Installing Factorio "+newVersion.IntToString()+".", glob.COLOR_CYAN)
		err = fullPackage(info)
		if err != nil {
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Status", "Install failed: "+err.Error(), glob.COLOR_CYAN))
			fact.FailOperation(opToken, "Updating Factorio", "Factorio install failed: "+err.Error(), glob.COLOR_RED)
			return info, fmt.Sprintf("DoQuickLatest: fullPackage: %v", err.Error()), true, false
		}
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Complete", "Factorio installed.", glob.COLOR_CYAN))
		fact.CompleteOperation(opToken, "Updating Factorio", "Factorio installed.", glob.COLOR_GREEN)
		return info, fmt.Sprintf("Factorio %v installed.", newVersion.IntToString()), false, false
	}

	err := GetFactorioVersion(info)
	if err != nil {
		fact.FailOperation(opToken, "Updating Factorio", "Unable to determine current Factorio version: "+err.Error(), glob.COLOR_RED)
		return info, fmt.Sprintf("Unable to get Factorio version: %v", err.Error()), true, false
	}
	oldVersion := info.VersInt

	newVersion, err := quickLatest(info)
	if err != nil {
		fact.FailOperation(opToken, "Updating Factorio", "Checking latest Factorio release failed: "+err.Error(), glob.COLOR_RED)
		return info, fmt.Sprintf("quickLatest: %v", err.Error()), true, false
	}
	fact.NewVersion = newVersion.IntToString()

	if isVersionNewerThan(*newVersion, oldVersion) || force {
		fact.UpdateOperation(opToken, "Updating Factorio", "Found Factorio update "+newVersion.IntToString()+". Downloading and installing.", glob.COLOR_CYAN)
		glob.SetUpdateMessage(disc.SmartWriteDiscordEmbed(cfg.Local.Channel.ChatChannel, &discordgo.MessageEmbed{Title: "Updating Factorio", Description: "Found Factorio update: " + newVersion.IntToString(), Color: glob.COLOR_CYAN}))

		info.VersInt = *newVersion
		err := fullPackage(info)
		if err != nil {
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Status", "Update failed: "+err.Error(), glob.COLOR_CYAN))
			fact.FailOperation(opToken, "Updating Factorio", "Factorio update failed: "+err.Error(), glob.COLOR_RED)
			return info, fmt.Sprintf("DoQuickLatest: fullPackage: %v", err.Error()), true, false
		}

		fact.CompleteOperation(opToken, "Updating Factorio", fmt.Sprintf("Factorio upgraded from %v to %v.", oldVersion.IntToString(), newVersion.IntToString()), glob.COLOR_GREEN)
		return info, fmt.Sprintf("Factorio upgraded from %v to %v.", oldVersion.IntToString(), newVersion.IntToString()), false, false
	} else if isVersionEqual(*newVersion, info.VersInt) {
		fact.CompleteOperation(opToken, "Updating Factorio", "Factorio is already up-to-date.", glob.COLOR_GREEN)
		return info, "Factorio is up-to-date.", false, true
	} else {
		fact.CompleteOperation(opToken, "Updating Factorio", "Current version is newer than the selected release channel.", glob.COLOR_GREEN)
		return info, "Current version is older, use install-factorio option if you want to downgrade.\nWARNING: downgrading will likely break existing maps/saves.", false, true
	}
}

func quickLatest(info *InfoData) (*versionInts, error) {

	body, _, err := HttpGet(false, releaseURL, true)
	if err != nil {
		return nil, errors.New("downloading release list failed: " + err.Error())
	}

	//Unmarshal into temporary list
	tempList := &getLatestData{}
	err = json.Unmarshal(body, &tempList)
	if err != nil {
		return nil, errors.New("unable to unmarshal json data: " + err.Error())
	}

	parseStringsLatest(tempList)
	if info.Xreleases {
		return &tempList.Experimental.HeadlessInt, err
	} else {
		return &tempList.Stable.HeadlessInt, err
	}
}
