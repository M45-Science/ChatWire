package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/util"
	"bytes"
	"encoding/json"
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
)

// Holy shit, this must be split up into smaller functions
func CheckModUpdates() (string, string, int) {

	if fact.FactorioVersion == constants.Unknown {
		cwlog.DoLogCW("CheckModUpdates: Factroio version unknown, aborting.")
		return "", "", 0
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
		emsg := "CheckModUpdates: Unable to read mods dir: " + err.Error()
		return emsg, emsg, 0
	}

	//Find all mods, read info.json inside
	var fileModList []modZipInfo
	for _, mod := range modList {
		if strings.HasSuffix(mod.Name(), ".zip") {
			modInfo := ModInfoRead(mod.Name(), nil)
			if modInfo == nil {
				continue
			}
			modInfo.oldFilename = mod.Name()
			fileModList = append(fileModList, *modInfo)
		}
	}

	//Read mods-list.json, continue even if it does not exist
	jsonFileList, _ := GetModList()

	//Check both lists, save any that are not explicitly disabled.
	var installedMods []modZipInfo
	found := false
	for _, fmod := range fileModList {
		//Check if the mod is disabled
		for _, jmod := range jsonFileList.Mods {
			if IsBaseMod(jmod.Name) {
				continue
			}

			if jmod.Name == fmod.Name && jmod.Enabled {
				found = true
				installedMods = append(installedMods, fmod)
				break
			}
		}
		//Include mods not found in mod-list.json
		if !found {
			installedMods = append(installedMods, fmod)
		}
	}

	//Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "The game has no installed mods to update."
		return emsg, emsg, 0
	}

	//Fetch mod portal data
	detailList := []modPortalFullData{}
	for _, item := range installedMods {
		URL := fmt.Sprintf(modPortalURL, item.Name)
		data, _, err := factUpdater.HttpGet(URL, true)
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
	for _, dItem := range detailList {
		for _, iItem := range installedMods {
			if dItem.Name == iItem.Name {
				found = false
				newestVersion := iItem.Version
				newestVersionData := ModReleases{}
				for _, release := range dItem.Releases {
					isNewer, err := newerVersion(newestVersion, release.Version)
					if err != nil {
						continue
					}
					if isNewer {
						//Check factorio version needed before adding
						for _, dep := range newestVersionData.InfoJSON.Dependencies {
							parts := strings.Split(dep, " ")
							numParts := len(parts)
							if strings.HasPrefix(dep, "base") {
								if numParts == 3 {
									tooNew, err := newerVersion(parts[2], fact.FactorioVersion)
									if err != nil {
										cwlog.DoLogCW("Unable to parse version: " + err.Error())
										continue
									}
									if !tooNew {
										cwlog.DoLogCW("Mod release: " + dItem.Name + ": " + release.Version + " requires a newer version of Factorio(" + parts[2] + "), skipping.")
										continue
									}
								}
							}
						}

						newestVersion = release.Version
						newestVersionData = release
						found = true
					}
				}
				if found {
					_, err = os.Stat(modPath + newestVersionData.FileName)
					if !os.IsNotExist(err) {
						//We don't need to download this, it already exists!
						cwlog.DoLogCW("The mod update " + newestVersionData.FileName + " already exists, skipping.")
						continue
					}

					if updatedCount != 0 {
						longBuf = longBuf + ", "
						shortBuf = shortBuf + ", "
					}
					updatedCount++

					downloadList = addDownload(
						downloadData{
							Name:        iItem.Name,
							Title:       iItem.Title,
							OldFilename: iItem.oldFilename,
							Data:        newestVersionData},
						downloadList)
				}
			}
		}
	}

	for d, dl := range downloadList {
		//Fetch the mod link
		dlSuffix := fmt.Sprintf(downloadSuffix, cfg.Global.Factorio.Username, cfg.Global.Factorio.Token)
		data, _, err := factUpdater.HttpGet(downloadPrefix+dl.Data.DownloadURL+dlSuffix, false)
		if err != nil {
			cwlog.DoLogCW("Unable to fetch URL: " + err.Error())
			continue
		}

		//Read the mod info.json
		zipIJ := ModInfoRead("", data)
		if zipIJ == nil {
			cwlog.DoLogCW("Mod download is invalid.")
			continue
		}

		//Check if the mod info.json looks correct
		if zipIJ.Name != dl.Name || zipIJ.Version != dl.Data.Version {
			cwlog.DoLogCW("Mod download failed verification.")
			continue
		}

		//Check mod for zip bomb
		if fact.BytesHasZipBomb(data) {
			cwlog.DoLogCW("Download contains zip bomb.")
			continue
		}

		//Write the new mod file as a temp file
		err = os.WriteFile(modPath+dl.Data.FileName+".tmp", data, 0755)
		if err != nil {
			cwlog.DoLogCW("Unable to write to mods directory: " + err.Error())
			continue
		}

		//Rename the temp file to the final name
		err = os.Rename(modPath+dl.Data.FileName+".tmp", modPath+dl.Data.FileName)
		if err != nil {
			cwlog.DoLogCW("Unable to rename temp file in mods directory: " + err.Error())
			continue
		}

		//Create old mods directory if needed
		_, err = os.Stat(modPath + constants.OldModsDir)
		if os.IsNotExist(err) {
			err = os.Mkdir(modPath+constants.OldModsDir, os.ModePerm)
			if err != nil {
				cwlog.DoLogCW("Unable to create old mods directory. " + err.Error())
				continue
			}
		}

		//Move old mod file into old directory
		if dl.OldFilename != "" {
			err = os.Rename(modPath+dl.OldFilename, modPath+constants.OldModsDir+"/"+dl.OldFilename)
			if err != nil {
				cwlog.DoLogCW("Unable to move old mod file in mods directory: " + err.Error())
				continue
			}
		} else {
			cwlog.DoLogCW("No old file found for: " + dl.Data.FileName)
		}

		downloadList[d].Complete = true

		mURL := fmt.Sprintf(displayURL, url.QueryEscape(dl.Name))
		longBuf = longBuf + "[" + dl.Title + "-" + dl.Data.Version + "](" + mURL + ")"
		shortBuf = shortBuf + dl.Name + "-" + dl.Data.Version
	}

	if updatedCount == 0 && len(installedMods) > 0 {
		emsg := "All installed mods are up to date."
		return emsg, emsg, 0
	} else if len(installedMods) > 0 {
		return shortBuf, longBuf, updatedCount
	} else {
		emsg := "There are no installed mods to update."
		return emsg, emsg, 0
	}
}

func addDownload(input downloadData, list []downloadData) []downloadData {
	for _, item := range list {
		if item.Data.FileName == input.Data.FileName && item.Data.DownloadURL == input.Data.DownloadURL {
			//Already exists, just return the list
			return list
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
