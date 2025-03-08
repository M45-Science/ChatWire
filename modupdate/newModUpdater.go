package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/util"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	modPortalURL   = "https://mods.factorio.com/api/mods/%v/full"
	displayURL     = "https://mods.factorio.com/mod/%v/changelog"
	downloadPrefix = "https://mods.factorio.com"
	downloadSuffix = "?username=%v&token=%v"
	modUpdateTitle = "Found Mod Updates"
)

// Holy shit, this must be split up into smaller functions
func CheckModUpdates(dryRun bool) (bool, error) {

	//If factorio failed to load, grab the version
	if fact.FactorioVersion == constants.Unknown {
		info := &factUpdater.InfoData{Xreleases: cfg.Local.Options.ExpUpdates, Build: "headless", Distro: "linux64"}
		factUpdater.GetFactorioVersion(info)
	}
	//Just in case that fails too
	if fact.FactorioVersion == constants.Unknown {
		emsg := "checkModUpdates: Factroio version unknown, aborting"
		return false, errors.New(emsg)
	}

	//Mod folder path
	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	//Read mods directory
	modList, err := os.ReadDir(modPath)
	if err != nil {
		emsg := "checkModUpdates: Unable to read mods dir: " + err.Error()
		return false, errors.New(emsg)
	}

	//Find all mods, read info.json inside each
	var modFileList []modZipInfo
	for _, mod := range modList {
		if strings.HasSuffix(mod.Name(), ".zip") {
			modInfo := ModInfoRead(mod.Name(), nil)
			if modInfo == nil {
				continue
			}
			modInfo.oldFilename = mod.Name()
			modFileList = append(modFileList, *modInfo)
		}
	}

	//Read mods-list.json, continue even if it does not exist
	jsonModList, _ := GetModList()

	//Check both lists, keep any that are not explicitly disabled.
	var installedMods []modZipInfo
	for _, modFile := range modFileList {
		//Check if the mod is disabled
		keep := true
		for _, jsonMod := range jsonModList.Mods {
			if IsBaseMod(jsonMod.Name) {
				continue
			}

			if strings.EqualFold(jsonMod.Name, modFile.Name) {
				if !jsonMod.Enabled {
					keep = false
				}
				break
			}
		}
		//Include mods not found in mod-list.json, unless disabled
		if keep {
			//Check for duplicates, then add
			dupe := false
			for _, item := range installedMods {
				//This shouldn't happen, but just in case
				if strings.EqualFold(item.Name, modFile.Name) &&
					item.Version == modFile.Version {
					dupe = true
					break
				}
			}
			if !dupe {
				installedMods = append(installedMods, modFile)
			}
		}
	}

	//Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "the game has no installed mods to update"
		return false, errors.New(emsg)
	}

	//Fetch mod portal data
	detailList := []modPortalFullData{}
	for _, item := range installedMods {
		URL := fmt.Sprintf(modPortalURL, item.Name)
		data, _, err := factUpdater.HttpGet(false, URL, true)
		if err != nil {
			cwlog.DoLogCW("Mod info request failed: " + err.Error())
			continue
		}
		newInfo := modPortalFullData{}
		err = json.Unmarshal(data, &newInfo)
		if err != nil {
			cwlog.DoLogCW("Mod info unmarshal failed: " + err.Error())
			continue
		}
		newInfo.oldFilename = item.oldFilename
		detailList = append(detailList, newInfo)
	}

	//Check mod postal data against mod list, find upgrades
	var downloadList []downloadData
	updatedCount := 0
	var shortBuf, longBuf string
	for _, portalItem := range detailList {
		for _, installedItem := range installedMods {
			if portalItem.Name == installedItem.Name {
				found := false
				candidate := installedItem.Version
				candidateData := ModReleases{}
				for _, release := range portalItem.Releases {
					//Check if this release is newer
					isNewer, err := checkVersion(EO_GREATER, candidate, release.Version)
					if err != nil {
						continue
					}
					//Check if factorio is new enough
					if isNewer {
						factReject, err := checkVersion(EO_GREATEREQ, fact.FactorioVersion, release.InfoJSON.FactorioVersion)
						if err != nil {
							cwlog.DoLogCW("Unable to parse version: " + err.Error())
							continue
						}
						if factReject {
							//cwlog.DoLogCW("Mod release: " + portalItem.Name + ": " + release.Version + " requires a different version of Factorio (" + release.InfoJSON.FactorioVersion + "), skipping.")
							continue
						}
						//Check dependencies
						reject := false
						for _, dep := range release.InfoJSON.Dependencies {
							//This flag isn't relevant
							dep = strings.TrimPrefix(dep, "~")
							dep = strings.TrimSpace(dep)

							//Optional dependency
							if strings.Contains(dep, "?") {
								continue
							}

							parts := strings.Split(dep, " ")
							numParts := len(parts)
							//Check base mods
							if IsBaseMod(parts[0]) {
								if numParts == 3 {
									eq := ParseOperator(parts[1])
									baseReject, err := checkVersion(eq, fact.FactorioVersion, parts[2])
									if err != nil {
										cwlog.DoLogCW("Unable to parse version: " + err.Error())
										reject = true
										continue
									}
									if baseReject {
										//cwlog.DoLogCW("Mod release: " + portalItem.Name + ": " + release.Version + " requires " + dep + " skipping.")
										reject = true
										continue
									}
								}
							}
							//Check if this dependency is already satisfied
							missingDep := false
							for _, mod := range installedMods {
								if strings.EqualFold(mod.Name, parts[0]) {

									//Warn about incompatable mods
									if strings.HasPrefix(dep, "!") {
										emsg := "Warning: " + mod.Name + " is incompatable with " + dep
										cwlog.DoLogCW(emsg)
										glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "WARNING", emsg, glob.COLOR_RED)
										continue
									}

									//If they specify a version for the dependency
									if numParts == 3 {
										eq := ParseOperator(parts[1])
										depGood, err := checkVersion(eq, mod.Version, parts[2])
										if err != nil {
											cwlog.DoLogCW("Unable to parse version: " + err.Error())
											missingDep = true
											continue
										}
										if !depGood {
											//cwlog.DoLogCW("Mod release: " + portalItem.Name + ": " + release.Version + " requires a different version of " + parts[0] + " (" + parts[2] + ").")
											missingDep = true
											continue
										}
									}
								}
							}

							if missingDep {
								//Process missing dependencies here
							}
						}

						//Save this release
						if !reject {
							candidate = release.Version
							candidateData = release
							found = true
						}
					}
				}
				if found {
					//Check if mod is already present before downloading
					_, err = os.Stat(modPath + candidateData.FileName)
					if !os.IsNotExist(err) {
						cwlog.DoLogCW("The mod update " + candidateData.FileName + " already exists, skipping.")
						continue
					}

					updatedCount++

					downloadList = addDownload(
						downloadData{
							Name:        installedItem.Name,
							Title:       installedItem.Title,
							OldFilename: installedItem.oldFilename,
							Data:        candidateData},
						downloadList)
				}
			}
		}
	}

	/*
	 * Check but do not download/install
	 * Used for debug
	 */
	if dryRun {
		for _, dl := range downloadList {
			cwlog.DoLogCW("%v-%v", dl.Name, dl.Data.Version)
		}
		return false, nil
	}

	//Show download status
	numDL := len(downloadList)
	if numDL > 0 {
		glob.UpdateMessage = nil
		buf := fmt.Sprintf("Downloading %v mod updates.", numDL)
		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, buf, glob.COLOR_CYAN)
	}
	//Show each download
	errorLog := ""
	for d, dl := range downloadList {
		buf := fmt.Sprintf("Downloading: %v-%v", dl.Name, dl.Data.Version)
		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, buf, glob.COLOR_CYAN)

		if errorLog != "" {
			glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, dl.Name+": "+errorLog, glob.COLOR_ORANGE)
			errorLog = ""
		}

		//Fetch the mod link
		dlSuffix := fmt.Sprintf(downloadSuffix, cfg.Global.Factorio.Username, cfg.Global.Factorio.Token)
		cwlog.DoLogCW("Downloading: " + dl.Data.DownloadURL)
		data, _, err := factUpdater.HttpGet(false, downloadPrefix+dl.Data.DownloadURL+dlSuffix, false)
		if err != nil {
			emsg := "Unable to fetch URL"
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		if !CheckSHA1(data, dl.Data.Sha1) {
			emsg := "Mod download is corrupted (invalid hash)."
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		//Read the mod info.json
		zipIJ := ModInfoRead("", data)
		if zipIJ == nil {
			emsg := "Mod download has invalid info.json."
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		//Check if the mod info.json looks correct
		if zipIJ.Name != dl.Name || zipIJ.Version != dl.Data.Version {
			emsg := "Mod download info.json failed verification."
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		//Check mod for zip bomb
		if fact.BytesHasZipBomb(data) {
			emsg := "Download contains possible zip bomb."
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		//Write the new mod file as a temp file
		err = os.WriteFile(modPath+dl.Data.FileName+".tmp", data, 0755)
		if err != nil {
			emsg := "Unable to write to mods directory"
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		//Rename the temp file to the final name
		err = os.Rename(modPath+dl.Data.FileName+".tmp", modPath+dl.Data.FileName)
		if err != nil {
			emsg := "Unable to rename temp file in mods directory"
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		//Create old mods directory if needed
		_, err = os.Stat(modPath + constants.OldModsDir)
		if os.IsNotExist(err) {
			err = os.Mkdir(modPath+constants.OldModsDir, os.ModePerm)
			if err != nil {
				emsg := "Unable to create old mods directory."
				cwlog.DoLogCW(emsg)
				errorLog = emsg
				continue
			}
		}

		//Move old mod file into old directory, if we had one
		if dl.OldFilename != "" {
			err = os.Rename(modPath+dl.OldFilename, modPath+constants.OldModsDir+"/"+dl.OldFilename)
			if err != nil {
				emsg := "Unable to move old mod file in mods directory"
				cwlog.DoLogCW(emsg)
				errorLog = emsg
				continue
			}
		} else {
			cwlog.DoLogCW("No old file found for: " + dl.Data.FileName)
		}

		downloadList[d].Complete = true

		if updatedCount != 0 {
			longBuf = longBuf + ", "
			shortBuf = shortBuf + ", "
		}
		mURL := fmt.Sprintf(displayURL, url.QueryEscape(dl.Name))
		longBuf = longBuf + "[" + dl.Title + "-" + dl.Data.Version + "](" + mURL + ")"
		shortBuf = shortBuf + dl.Name + "-" + dl.Data.Version
	}

	//TO DO: Report error, don't report all up to date with errors
	if updatedCount > 0 && len(installedMods) > 0 {
		emsg := "Mod updates complete."
		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Mod Updates", emsg, glob.COLOR_CYAN)
		return true, nil
	}

	return false, errors.New("No mod updates available.")
}

func addDownload(input downloadData, list []downloadData) []downloadData {
	for i, item := range list {
		if strings.EqualFold(item.Name, input.Name) {
			//Check versions
			newer, err := checkVersion(EO_GREATER, item.Data.Version, input.Data.Version)
			if err != nil {
				cwlog.DoLogCW("addDownload: Unable to parse version")
				return list
			}
			if newer {
				//Already in list and not newer, skip
				return list
			} else {
				//Already here, but older. Replace it
				list[i] = input
			}
		}
	}

	return append(list, input)
}

func ConfigGameMods(controlList []string, setState bool) (*modListData, error) {
	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/" + constants.ModListName

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	serverMods := modListData{}
	err = json.Unmarshal(data, &serverMods)
	if err != nil {
		return nil, err
	}

	if len(controlList) > 0 {
		for s, serverMod := range serverMods.Mods {
			if strings.EqualFold(serverMod.Name, "base") {
				continue
			}
			for _, controlMod := range controlList {
				if strings.EqualFold(serverMod.Name, controlMod) {
					serverMods.Mods[s].Enabled = setState

					cwlog.DoLogCW(util.BoolToString(setState) + " " + serverMod.Name)
				}
			}
		}

		outbuf := new(bytes.Buffer)
		enc := json.NewEncoder(outbuf)
		enc.SetIndent("", "\t")

		if err := enc.Encode(serverMods); err != nil {
			return nil, err
		}

		err = os.WriteFile(path, outbuf.Bytes(), 0644)
		cwlog.DoLogCW("Wrote " + constants.ModListName)
	}
	return &serverMods, err
}
