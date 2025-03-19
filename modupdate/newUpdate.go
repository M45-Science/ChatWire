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

const (
	modPortalURL   = "https://mods.factorio.com/api/mods/%v/full"
	displayURL     = "https://mods.factorio.com/mod/%v/changelog"
	downloadPrefix = "https://mods.factorio.com"
	downloadSuffix = "?username=%v&token=%v"
	modUpdateTitle = "Found Mod Updates"
)

/* Read entire mod folder */
func CheckMods(force bool, reportNone bool) {

	if !cfg.Local.Options.ModUpdate && !force {
		return
	}

	updated, err := CheckModUpdates(false)
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

const resolveDepsDebug = false

func resolveDeps(modPortalData []modPortalFullData, wasDep bool) ([]downloadData, error) {
	var downloadMods []downloadData
	for _, item := range modPortalData {

		//Find a release that fits
		candidate := modRelease{Version: "0.0.0"}
		if item.installed.Version != "" {
			candidate.Version = item.installed.Version
		}

		for _, rel := range item.Releases {
			//cwlog.DoLogCW("RELEASES: %v: Local: %v, Rel: %v", item.Name, item.installed.Version, rel.Version)

			releaseNewer := false
			if item.installed.Version == "" {
				releaseNewer = true
			} else {
				var err error
				releaseNewer, err = checkVersion(EO_GREATER, item.installed.Version, rel.Version)
				if err != nil {
					return []downloadData{}, err
				}
			}
			if releaseNewer {
				if resolveDepsDebug {
					cwlog.DoLogCW("NEWER: %v: LOCAL: %v, Rel: %v", item.Name, item.installed.Version, rel.Version)
				}
				candidateNewer, err := checkVersion(EO_GREATER, candidate.Version, rel.Version)
				if err != nil {
					return []downloadData{}, err
				}
				if candidateNewer {
					depsMet := true
					for _, dep := range rel.InfoJSON.Dependencies {
						depInfo := parseDep(dep)
						if depInfo.incompatible {
							continue
						}
						if depInfo.optional {
							continue
						}
						//Check base mod version
						if resolveDepsDebug {
							cwlog.DoLogCW("dep name: %v, eq: %v, vers: %v :: inc: %v", depInfo.name, operatorToString(depInfo.equality), depInfo.version, depInfo.incompatible)
						}
						if IsBaseMod(depInfo.name) {
							if depInfo.version != "" {
								good, err := checkVersion(depInfo.equality, depInfo.version, fact.FactorioVersion)
								if !good || err != nil {
									depsMet = false
									continue
								}
							}
							if resolveDepsDebug {
								cwlog.DoLogCW("base dep available: %v", dep)
							}
						} else {
							haveDepInfo := false
							depPortalInfo := modPortalFullData{}
							if resolveDepsDebug {
								cwlog.DoLogCW("CHECKING DEP %v-%v", depInfo.name, depInfo.version)
							}
							for _, item := range modPortalData {
								if item.Name == depInfo.name {
									haveDepInfo = true
									depPortalInfo = item
									break
								}
							}
							if !haveDepInfo {
								depPortalInfo, err = DownloadModInfo(depInfo.name)
								if err != nil {
									cwlog.DoLogCW("resolveDeps: dep: DownloadModInfo: %v", err)
									return []downloadData{}, err
								}
							}
							dl, err := resolveDeps([]modPortalFullData{depPortalInfo}, true)
							if err != nil {
								cwlog.DoLogCW("resolveDeps: dep: resolveDeps: %v", err)
								return []downloadData{}, err
							}
							if len(dl) > 0 {
								for _, item := range dl {
									downloadMods = addDownload(item, downloadMods)
								}
							}
						}
					}
					if depsMet {
						candidate = rel
					}
				}
			}
		}

		if candidate.Version != "0.0.0" && item.installed.Version != candidate.Version {
			downloadMods = addDownload(downloadData{Title: item.Title, Name: item.Name, Filename: candidate.FileName,
				OldFilename: item.installed.Filename, Data: candidate, Version: candidate.Version,
				OldVersion: item.installed.Version, wasDep: wasDep,
			}, downloadMods)
		}

	}

	return downloadMods, nil
}

func CheckModUpdates(dryRun bool) (bool, error) {
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
		//cwlog.DoLogCW("Getting portal info: %v", item.Name)
		newInfo, err := DownloadModInfo(item.Name)
		if err != nil {
			cwlog.DoLogCW("NEWCheckModUpdates: DownloadModInfo" + err.Error())
			return false, err
		}

		newInfo.filename = item.Filename
		newInfo.installed = item
		modPortalData = append(modPortalData, newInfo)
		//cwlog.DoLogCW("Got portal info: %v", newInfo.Name)
	}

	downloadList, err := resolveDeps(modPortalData, false)
	if err != nil {
		cwlog.DoLogCW("NEWCheckModUpdates: resolveDeps: " + err.Error())
		return false, err
	}

	//Dry run ends here
	if dryRun {
		for _, dl := range downloadList {
			cwlog.DoLogCW("%v-%v: %v", dl.Name, dl.Data.Version, dl.Filename)
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

	return depRequires{name: name, equality: parseOperator(equality), version: version, optional: optional, incompatible: incompatible}
}
