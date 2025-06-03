package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	modPortalURL   = "https://mods.factorio.com/api/mods/%v/full"
	displayURL     = "https://mods.factorio.com/mod/%v/changelog"
	downloadPrefix = "https://mods.factorio.com"
	downloadSuffix = "?username=%v&token=%v"
	modUpdateTitle = "Found Mod Updates"
)

func CheckMods(force bool, reportNone bool) {

	if !cfg.Local.Options.ModUpdate && !force {
		return
	}

	updated, err := CheckModUpdates(false)
	if reportNone || updated {
		if err != nil {
			glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Warning:", err.Error(), glob.COLOR_CYAN)
		}
	}
	if updated && fact.FactIsRunning {
		fact.QueueFactReboot = true
	}
}

const resolveDepsDebug = false

func resolveDeps(modPortalData []modPortalFullData, wasDep bool, depth int) ([]downloadData, error) {

	if depth > 10 {
		return []downloadData{}, nil
	}

	var downloadMods []downloadData
	for _, item := range modPortalData {

		//Don't follow circular deps
		circular := false
		depParentsLock.Lock()
		for _, parent := range depParents {
			if item.Name == parent {
				circular = true
				break
			}
		}
		depParentsLock.Unlock()
		if circular {
			continue
		}

		candidate := modRelease{Version: "0.0.0"}
		if item.installed.Version != "" {
			candidate.Version = item.installed.Version
		}

		//Check all releases
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
			//If release is newer, check candidate
			if releaseNewer {
				if resolveDepsDebug {
					cwlog.DoLogCW("NEWER: %v: LOCAL: %v, Rel: %v", item.Name, item.installed.Version, rel.Version)
				}
				releaseNewer, err := checkVersion(EO_GREATER, candidate.Version, rel.Version)
				if err != nil {
					return []downloadData{}, err
				}
				//If release is newer check deps
				if releaseNewer {
					depsMet := true

					for _, dep := range rel.InfoJSON.Dependencies {
						depInfo := parseDep(dep)
						if depInfo.incompatible {
							continue
						}
						//We can ignore optional deps
						if depInfo.optional {
							continue
						}

						//Check base mod version
						if resolveDepsDebug {
							cwlog.DoLogCW("dep name: %v, eq: %v, vers: %v :: inc: %v", depInfo.name, operatorToString(depInfo.equality), depInfo.version, depInfo.incompatible)
						}
						//If dep is a base mod, check it here
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
						} else { //Dep is a mod, check if we have it
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
							//We do not have the dep, download info
							if !haveDepInfo {
								depPortalInfo, err = DownloadModInfo(depInfo.name)
								if err != nil {
									cwlog.DoLogCW("resolveDeps: dep: DownloadModInfo: %v", err)
									return []downloadData{}, err
								}
							}
							//Recursively check dep's deps
							depParentsLock.Lock()
							depParents = append(depParents, item.Name)
							depParentsLock.Unlock()
							dl, err := resolveDeps([]modPortalFullData{depPortalInfo}, true, depth+1)
							if err != nil {
								cwlog.DoLogCW("resolveDeps: dep: resolveDeps: %v", err)
								return []downloadData{}, err
							}
							//Download dep and all of dep's deps.
							if len(dl) > 0 {
								for _, item := range dl {
									downloadMods = addDownload(item, downloadMods)
								}
							}
						}
					}
					//If deps were met, we can update the candidate
					if depsMet {
						candidate = rel
					}
				}
			}
		}

		//Add candidate to the download list
		if candidate.Version != "0.0.0" && item.installed.Version != candidate.Version {
			downloadMods = addDownload(downloadData{Title: item.Title, Name: item.Name, Filename: candidate.FileName,
				OldFilename: item.installed.Filename, Data: candidate, Version: candidate.Version,
				OldVersion: item.installed.Version, wasDep: wasDep,
			}, downloadMods)
		}

	}

	//Return list of downloads
	return downloadMods, nil
}

var depParents []string
var depParentsLock sync.Mutex

func CheckModUpdates(dryRun bool) (bool, error) {
	// If needed, get Factorio version
	getFactoioVersion()

	// Read all mod.zip files
	modFileList, err := GetModFiles()
	if err != nil {
		return false, err
	}

	// Read mods-list.json
	jsonModList, _ := GetModList() //Ignore error, mods-list.json missing isn't the end of the world.
	// Merge the two lists
	installedMods := MergeModLists(modFileList, jsonModList)

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

		//Save the filename, for dealing with old versions later
		newInfo.filename = item.Filename
		newInfo.installed = item
		modPortalData = append(modPortalData, newInfo)
		//cwlog.DoLogCW("Got portal info: %v", newInfo.Name)
	}

	downloadList, err := resolveDeps(modPortalData, false, 0)
	depParentsLock.Lock()
	depParents = []string{}
	depParentsLock.Unlock()

	if err != nil {
		cwlog.DoLogCW("NEWCheckModUpdates: resolveDeps: " + err.Error())
		return false, err
	}

	_, err = checkIncompatible(installedMods, downloadList)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return false, err
	}

	//Dry run ends here
	if dryRun {
		for _, dl := range downloadList {
			cwlog.DoLogCW("%v-%v: %v", dl.Name, dl.Data.Version, dl.Filename)
		}
		return false, nil
	}

	if len(downloadList) > 0 {
		newHist := ModHistoryItem{Name: "Mod update started", InfoItem: true, Date: time.Now()}
		AddModHistory(newHist)
	}
	shortBuf := downloadMods(downloadList)

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

type incMod struct {
	Name, Version string
	Deps          []string
}

func checkIncompatible(installed []modZipInfo, downloadList []downloadData) (bool, error) {

	combined := []incMod{}
	for _, imod := range installed {
		if !imod.Enabled {
			continue
		}
		combined = append(combined, incMod{Name: imod.Name, Version: imod.Version, Deps: imod.Dependencies})
	}
	for _, dmod := range downloadList {
		combined = append(combined, incMod{Name: dmod.Name, Version: dmod.Version, Deps: dmod.Data.InfoJSON.Dependencies})
	}

	for _, itemA := range combined {
		for _, itemB := range combined {
			if itemA.Name == itemB.Name {
				continue
			}
			for _, depA := range itemA.Deps {
				depInfo := parseDep(depA)
				if depInfo.optional {
					continue
				}
				if depInfo.incompatible {
					if itemB.Name == depInfo.name {
						good, _ := checkVersion(depInfo.equality, depInfo.version, itemB.Version)
						if !good {
							emsg := fmt.Sprintf("checkIncompatible: %v-%v not compatible with %v-%v (%v)! Auto-update disabled.", itemA.Name, itemA.Version, itemB.Name, itemB.Version, depInfo.name)
							cwlog.DoLogCW(emsg)
							cfg.Local.Options.ModUpdate = false
							cfg.WriteLCfg()
							return true, errors.New(emsg)
						}
					}
				}
			}
		}
	}

	return false, nil
}

func parseDep(input string) depRequires {
	incompatible, optional := false, false

	input = strings.TrimSpace(input)
	//Mark incompatible
	if strings.ContainsAny(input, "!") {
		incompatible = true
	}
	//Mark optional
	if strings.HasPrefix(input, "?") || strings.HasPrefix(input, "(?)") || strings.HasPrefix(input, "( ? )") {
		optional = true
	}
	//Remove prefixes before processing
	input = strings.TrimPrefix(input, "~")
	input = strings.TrimPrefix(input, "!")
	input = strings.TrimPrefix(input, "?")
	input = strings.TrimPrefix(input, "(?)")
	input = strings.TrimPrefix(input, "( ? )")
	input = strings.TrimSpace(input)

	nameEnd := 0
	versionStart := 0
	// This properly handles malformed dependencies (no spaces) *cough flare stack cough*
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
