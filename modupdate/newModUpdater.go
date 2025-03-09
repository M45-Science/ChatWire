package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"errors"
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
	installedMods := MergeModLists(modFileList, jsonModList)

	//Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "the game has no installed mods to update"
		return false, errors.New(emsg)
	}

	//Fetch mod portal data
	detailList := []ModPortalFullData{}
	for _, item := range installedMods {
		newInfo, _ := downloadModInfo(item.Name)
		newInfo.oldFilename = item.OldFilename
		detailList = append(detailList, newInfo)
	}

	//Make a list of mods to download/check
	downloadList := findModUpgrades(installedMods, detailList)

	//Check for mod dependencies
	downloadList = checkModDependencies(downloadList)

	//Dry run ends here
	if dryRun {
		for _, dl := range downloadList {
			cwlog.DoLogCW("%v-%v: %v", dl.Name, dl.Data.Version, dl.doDownload)
		}
		return false, nil
	}

	downloadMods(downloadList)

	//TO DO: Report error, don't report all up to date with errors
	if getDownloadCount(downloadList) > 0 && len(installedMods) > 0 {
		emsg := "Mod updates complete."
		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Mod Updates", emsg, glob.COLOR_CYAN)
		return true, nil
	}

	return false, errors.New("No mod updates available.")
}
