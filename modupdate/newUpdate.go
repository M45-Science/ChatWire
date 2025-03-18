package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"errors"
	"strings"
)

/* Read entire mod folder */
func NEWCheckMods(force bool, reportNone bool) {

	if !cfg.Local.Options.ModUpdate && !force {
		return
	}

	updated, err := NEWCheckModUpdates(false)
	if reportNone {
		title := modUpdateTitle
		buf := ""
		if err != nil {
			buf = err.Error()
			title = "Error"
		}
		if buf != "" {
			glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, title, buf, glob.COLOR_CYAN)
		}
	}
	if updated && err == nil {
		if fact.FactIsRunning {
			fact.QueueFactReboot = true
		}
	}
}

func NEWCheckModUpdates(dryRun bool) (bool, error) {
	// If needed, get Factorio version
	getFactoioVersion()

	// Read all mod.zip files
	modFileList, err := GetModFiles()
	if err != nil {
		return false, err
	}

	// Read mods-list.json
	jsonModList, _ := GetModList()
	// Merge the two lists
	installedMods := mergeModLists(modFileList, jsonModList)

	// Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "the game has no installed mods to update"
		return false, errors.New(emsg)
	}

	// Fetch mod portal data
	modPortalData := []modPortalFullData{}
	for _, item := range installedMods {
		if IsBaseMod(item.Name) {
			continue
		}
		cwlog.DoLogCW("Getting portal info: %v", item.Name)
		newInfo, err := DownloadModInfo(item.Name)
		if err != nil {
			cwlog.DoLogCW("NEWCheckModUpdates: DownloadModInfo" + err.Error())
			return false, err
		}

		newInfo.filename = item.Filename
		newInfo.installed = item
		modPortalData = append(modPortalData, newInfo)
		cwlog.DoLogCW("Got portal info: %v", newInfo.Name)
	}

	for _, item := range modPortalData {
		//Find a release that fits
		candidate := modRelease{Version: "0.0.0"}
		for _, rel := range item.Releases {
			//cwlog.DoLogCW("%v: Local: %v, Rel: %v", item.Name, item.installed.Version, rel.Version)
			goodA, err := checkVersion(EO_GREATEREQ, item.installed.Version, rel.Version)
			if err != nil {
				return false, err
			}
			if goodA {
				cwlog.DoLogCW("%v: Cand: %v, Rel: %v", item.Name, candidate.Version, rel.Version)
				goodB, err := checkVersion(EO_GREATEREQ, candidate.Version, rel.Version)
				if err != nil {
					return false, err
				}
				if goodB {
					for _, dep := range rel.InfoJSON.Dependencies {
						depInfo := parseDep(dep)
						if depInfo.optional {
							continue
						}
						cwlog.DoLogCW("name: %v, eq: %v, vers: %v :: inc: %v", depInfo.name, depInfo.equality, depInfo.version, depInfo.incompatible)
					}
				}
			}
		}
	}

	downloadList := []downloadData{}

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

func parseDep(input string) depRequires {
	incompatible, optional := false, false

	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "!") {
		incompatible = true
	}
	if strings.HasPrefix(input, "?") || strings.HasPrefix(input, "(?)") || strings.HasPrefix(input, "( ? )") {
		optional = true
	}
	input = strings.TrimPrefix(input, "~")
	input = strings.TrimPrefix(input, "!")
	input = strings.TrimPrefix(input, "?")
	input = strings.TrimPrefix(input, "(?)")
	input = strings.TrimPrefix(input, "( ? )")
	input = strings.TrimSpace(input)

	nameEnd := 0
	versionStart := 0
	for c, ch := range input {
		if ch == '>' || ch == '<' || ch == '=' {
			if nameEnd == 0 {
				nameEnd = c
			}
			versionStart = c
		}
	}
	if nameEnd == 0 {
		return depRequires{name: input, optional: optional, incompatible: incompatible}
	}
	name := strings.TrimSpace(input[:nameEnd])
	equality := strings.TrimSpace(input[nameEnd : versionStart+1])
	version := strings.TrimSpace(input[versionStart+1:])

	return depRequires{name: name, equality: equality, version: version, optional: optional, incompatible: incompatible}
}
