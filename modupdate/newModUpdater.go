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

	//Check both lists, keep any that are not explicitly disabled.
	var installedMods []modZipInfo
	for _, fmod := range fileModList {
		//Check if the mod is disabled
		doAdd := true
		for _, jmod := range jsonFileList.Mods {
			if IsBaseMod(jmod.Name) {
				continue
			}

			if strings.EqualFold(jmod.Name, fmod.Name) {
				if !jmod.Enabled {
					doAdd = false
				}
				break
			}
		}
		//Include mods not found in mod-list.json, unless disabled
		if doAdd {
			dupe := false
			for _, item := range installedMods {
				if strings.EqualFold(item.Name, fmod.Name) {
					dupe = true
					break
				}
			}
			if !dupe {
				installedMods = append(installedMods, fmod)
			}
		}
	}

	//Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "the game has no installed mods to update"
		return false, errors.New(emsg)
	}

	//Satisfy dependencies
	for _, item := range installedMods {
		for _, dep := range item.Dependencies {
			//Skip base mods
			if IsBaseMod(dep) {
				continue
			}

			//Skip optional deps
			if strings.HasPrefix(dep, "?") || strings.HasPrefix(dep, "(?)") {
				continue
			}

			//Remove order flag
			dep = strings.TrimPrefix(dep, "~")

			//Split into parts for version equality operators
			depParts := strings.Split(dep, " ")
			numDepParts := len(depParts)

			depStr := dep
			depVers := ""
			if numDepParts == 3 {
				depStr = depParts[0]
				depVers = depParts[2]
			}
			if depVers != "" {
				//placeholder
			}

			depFound := false
			for _, item := range installedMods {
				if item.Name == depStr {
					depFound = true
					break
				}
			}
			if !depFound {
				cwlog.DoLogCW("Need dep: " + depStr)
				//Maybe just sync mods for ease?
			}
		}

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
	for _, dItem := range detailList {
		for _, iItem := range installedMods {
			if dItem.Name == iItem.Name {
				found := false
				newestVersion := iItem.Version
				newestVersionData := ModReleases{}
				for _, release := range dItem.Releases {
					//Check if this release is newer
					isNewer, err := checkVersion(EO_LESS, newestVersion, release.Version)
					if err != nil {
						continue
					}
					//Check if factorio is new enough
					if isNewer {
						rejectFact, err := checkVersion(EO_LESS, fact.FactorioVersion, release.InfoJSON.FactorioVersion)
						if err != nil {
							cwlog.DoLogCW("Unable to parse version: " + err.Error())
							continue
						}
						if rejectFact {
							cwlog.DoLogCW("Mod release: " + dItem.Name + ": " + release.Version + " requires a different version of Factorio (" + release.InfoJSON.FactorioVersion + "), skipping.")
							continue
						}
						//Check base mod version needed
						reject := false
						for _, dep := range release.InfoJSON.Dependencies {
							parts := strings.Split(dep, " ")
							numParts := len(parts)
							if strings.HasPrefix(dep, "base") {
								//If they specify a base version
								if numParts == 3 {
									eq := ParseEquality(parts[1])
									rejectBase, err := checkVersion(eq, parts[2], fact.FactorioVersion)
									if err != nil {
										cwlog.DoLogCW("Unable to parse version: " + err.Error())
										reject = true
										continue
									}
									if rejectBase {
										cwlog.DoLogCW("Mod release: " + dItem.Name + ": " + release.Version + " requires a different version of base (" + parts[2] + "), skipping.")
										reject = true
										continue
									}
								}
							}
						}

						//Save this release
						if !reject {
							newestVersion = release.Version
							newestVersionData = release
							found = true
						}
					}
				}
				if found {
					_, err = os.Stat(modPath + newestVersionData.FileName)
					if !os.IsNotExist(err) {
						//We don't need to download this, it already exists!
						cwlog.DoLogCW("The mod update " + newestVersionData.FileName + " already exists, skipping.")
						continue
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

	//Check but do not download/install
	if dryRun {
		for _, dl := range downloadList {
			cwlog.DoLogCW("%v-%v", dl.Name, dl.Data.Version)
		}
		return false, nil
	}

	numDL := len(downloadList)
	if numDL > 0 {
		glob.UpdateMessage = nil
		buf := fmt.Sprintf("Downloading %v mod updates.", numDL)
		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, buf, glob.COLOR_CYAN)
	}
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

		//Move old mod file into old directory
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
