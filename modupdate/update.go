package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"errors"
	"time"
)

const (
	modPortalURL   = "https://mods.factorio.com/api/mods/%v/full"
	displayURL     = "https://mods.factorio.com/mod/%v/changelog"
	downloadPrefix = "https://mods.factorio.com"
	downloadSuffix = "?username=%v&token=%v"
	modUpdateTitle = "Found Mod Updates"
)

func CheckModUpdates(dryRun bool) (bool, error) {

	//If needed, get Factorio version
	getFactoioVersion()

	//Read all mod.zip files
	modFileList, err := GetModFiles()
	if err != nil {
		return false, err
	}
	//Read mods-list.json
	jsonModList, _ := GetModList()
	//Merge the two lists
	installedMods := mergeModLists(modFileList, jsonModList)

	//Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "the game has no installed mods to update"
		return false, errors.New(emsg)
	}

	//Fetch mod portal data
	detailList := []modPortalFullData{}
	for _, item := range installedMods {
		if IsBaseMod(item.Name) {
			continue
		}
		newInfo, _ := DownloadModInfo(item.Name)
		newInfo.filename = item.Filename
		detailList = append(detailList, newInfo)
	}

	//Make a list of mods to download/check
	downloadList := findModUpgrades(installedMods, detailList)

	//Check for mod dependencies
	downloadList, err = resolveModDependencies(downloadList)

	if err != nil {
		//glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Mod Updates: Error", err.Error(), glob.COLOR_ORANGE)
		cwlog.DoLogCW("CheckModUpdates: Error: " + err.Error())
		time.Sleep(constants.ErrMsgDelay)
	}

	//Dry run ends here
	if dryRun {
		for _, dl := range downloadList {
			cwlog.DoLogCW("%v-%v: %v", dl.Name, dl.Data.Version, dl.doDownload)
		}
		return false, nil
	}

	shortBuf := downloadMods(downloadList)

	//TO DO: Report error, don't report all up to date with errors
	if getDownloadCount(downloadList) > 0 && len(installedMods) > 0 {
		emsg := "Mod updates complete."
		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Mod Updates", emsg, glob.COLOR_CYAN)
		if fact.NumPlayers > 0 && shortBuf != "" {
			fact.FactChat("Mod updates: " + shortBuf + " (Mods will update on reboot, when server is empty)")
		}
		glob.BootMessage = nil
		return true, nil
	}

	glob.BootMessage = nil
	return false, errors.New("No mod updates available.")
}

/* Read entire mod folder */
func CheckMods(force bool, reportNone bool) {

	if !cfg.Local.Options.AutoUpdate && !force {
		return
	}

	updated, err := CheckModUpdates(false)
	if reportNone {
		buf := ""
		if err != nil {
			buf = err.Error()
		}
		if buf != "" {
			glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, buf, glob.COLOR_CYAN)
		}
	}
	if updated && err == nil {
		if fact.FactIsRunning {
			fact.QueueFactReboot = true
		}
	}
}
