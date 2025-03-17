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
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const modHistoryFile = "modUpdateHistory.dat"

func WriteModHistory() bool {
	tempPath := modHistoryFile + ".tmp"
	finalPath := modHistoryFile

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(ModHistory); err != nil {
		cwlog.DoLogCW("writeModHistory: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("writeModHistory: os.Create failure")
		return false
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("writeModHistory: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("writeModHistory't rename modHistory file.")
		return false
	}

	return true
}

func ReadModHistory() bool {

	file, err := os.ReadFile(modHistoryFile)

	if file != nil && err == nil {
		newHist := ModHistoryData{}

		err := json.Unmarshal([]byte(file), &newHist)
		if err != nil {
			cwlog.DoLogCW("readModHistory: Unmarshal failure")
			cwlog.DoLogCW(err.Error())
			return false
		}

		ModHistory = newHist

		return true
	} else {
		cwlog.DoLogCW("readModHistory: ReadFile failure")
		return false
	}
}

func checkVersion(operator int, local, remote string) (bool, error) {

	cInt, err := versionToInt(local)
	if err != nil {
		return false, err
	}
	rInt, err := versionToInt(remote)
	if err != nil {
		return false, err
	}

	// Compare major versions
	if rInt.parts[0] != cInt.parts[0] {
		return compareVersions(operator, cInt.parts[0], rInt.parts[0])
	}

	// Compare minor versions
	if rInt.parts[1] != cInt.parts[1] {
		return compareVersions(operator, cInt.parts[1], rInt.parts[1])
	}

	// Compare patch versions
	if rInt.parts[2] != cInt.parts[2] {
		return compareVersions(operator, cInt.parts[2], rInt.parts[2])
	}

	// If they are equal
	return compareVersions(operator, 0, 0)
}

// Helper function to compare based on eo
func compareVersions(eo, av, bv int) (bool, error) {
	switch eo {
	case EO_LESS:
		return bv < av, nil
	case EO_LESSEQ:
		return bv <= av, nil
	case EO_EQUAL:
		return bv == av, nil
	case EO_GREATEREQ:
		return bv >= av, nil
	case EO_GREATER:
		return bv > av, nil
	default:
		return false, errors.New("invalid comparison operation")
	}
}

func versionToInt(data string) (intVersion, error) {
	parts := strings.Split(data, ".")
	numParts := len(parts)
	//For 2 digit versions
	if numParts == 2 {
		data = data + ".0"
		numParts++
	}

	var intOut intVersion
	if numParts != 3 {
		return intVersion{}, errors.New("malformed version string: " + data)
	}
	for p, part := range parts {
		val, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return intVersion{}, errors.New("failed to parse version string")
		}
		intOut.parts[p] = int(val)
	}
	return intOut, nil
}

func IsBaseMod(dep string) bool {
	//Add detection of equality operators
	if dep == "base" ||
		dep == "elevated-rails" ||
		dep == "quality" ||
		dep == "space-age" {
		return true
	}
	return false
}

func IsBasePrefix(dep string) bool {
	//Add detection of equality operators
	if strings.HasPrefix(dep, "base") ||
		strings.HasPrefix(dep, "elevated-rails") ||
		strings.HasPrefix(dep, "quality") ||
		strings.HasPrefix(dep, "space-age") {
		return true
	}
	return false
}
func GetModList() (ModListData, error) {
	path := util.GetModsFolder() + constants.ModListName

	data, err := os.ReadFile(path)
	if err != nil {
		return ModListData{}, err
	}

	serverMods := ModListData{}
	err = json.Unmarshal(data, &serverMods)
	if err != nil {
		return ModListData{}, err
	}

	return serverMods, nil
}

func modInfoRead(modName string, rawData []byte) *modZipInfo {
	var err error
	if rawData == nil {
		path := util.GetModsFolder() + "/" + modName

		rawData, err = os.ReadFile(path)
		if err != nil {
			cwlog.DoLogCW("ReadModZipInfo: " + err.Error())
			return nil
		}
	}
	jsonData := getModJSON(rawData)

	modData := modZipInfo{}
	err = json.Unmarshal(jsonData, &modData)
	if err != nil {
		cwlog.DoLogCW("ReadModZipInfo: Unmarshal failure: " + err.Error())
		buf := fmt.Sprintf("%v", modData)
		cwlog.DoLogCW(buf)
		return nil
	}

	return &modData
}

func getModJSON(data []byte) []byte {
	// Create a reader from the byte array
	byteReader := bytes.NewReader(data)

	// Create a zip reader
	zipReader, err := zip.NewReader(byteReader, int64(len(data)))
	if err != nil {
		return nil
	}

	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "info.json") {
			if strings.Count(file.Name, "/") < 2 {
				f, err := file.Open()
				if err != nil {
					return nil
				}
				defer f.Close()

				var buf bytes.Buffer
				_, err = io.Copy(&buf, f)
				if err != nil {
					return nil
				}

				return buf.Bytes()
			}
		}
	}

	return nil
}

func parseOperator(input string) int {
	switch input {
	case "<":
		return EO_LESS
	case "<=":
		return EO_LESSEQ
	case "=":
		return EO_EQUAL
	case ">=":
		return EO_GREATEREQ
	case ">":
		return EO_GREATER
	default:
		return EO_ERROR
	}
}

func checkSHA1(data []byte, checkHash string) bool {

	hash := sha1.New()
	hash.Write([]byte(data))
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return strings.EqualFold(hashString, checkHash)
}

func GetModFiles() ([]modZipInfo, error) {
	//Read mods directory
	modList, err := os.ReadDir(util.GetModsFolder())
	if err != nil {
		emsg := "checkModUpdates: Unable to read mods dir: " + err.Error()
		return nil, errors.New(emsg)
	}

	//Find all mods, read info.json inside each
	var modFileList []modZipInfo
	for _, mod := range modList {
		if strings.HasSuffix(mod.Name(), ".zip") {
			modInfo := modInfoRead(mod.Name(), nil)
			if modInfo == nil {
				continue
			}
			modInfo.Filename = mod.Name()
			modFileList = append(modFileList, *modInfo)
		}
	}

	return modFileList, nil
}

func mergeModLists(modFileList []modZipInfo, jsonModList ModListData) []modZipInfo {
	//Check both lists, keep any that are not explicitly disabled.
	var installedMods []modZipInfo
	for _, modFile := range modFileList {
		dupe := false
		for _, item := range installedMods {
			if item.Name == modFile.Name {
				dupe = true
				break
			}
		}
		if !dupe {
			installedMods = append(installedMods, modFile)
		}
	}
	for _, modFile := range jsonModList.Mods {
		dupe := false
		for _, item := range installedMods {
			//This shouldn't happen, but just in case
			if item.Name == modFile.Name {
				dupe = true
				break
			}
		}
		if !dupe {
			installedMods = append(installedMods, modZipInfo{Name: modFile.Name, Version: "0.0.0"})
		}
	}

	return installedMods
}

func getFactoioVersion() {
	//If factorio failed to load, grab the version
	if fact.FactorioVersion == constants.Unknown {
		info := &factUpdater.InfoData{Xreleases: cfg.Local.Options.ExpUpdates, Build: "headless", Distro: "linux64"}
		factUpdater.GetFactorioVersion(info)
		fact.FactorioVersion = info.VersInt.IntToString()
	}
	//Just in case that fails too
	if fact.FactorioVersion == constants.Unknown {
		emsg := "checkModUpdates: Factroio version unknown, aborting"
		cwlog.DoLogCW(emsg)
	}
}

func findModUpgrades(installedMods []modZipInfo, detailList []modPortalFullData) []downloadData {
	//Check mod postal data against mod list, find upgrades
	var downloadList []downloadData
	for _, installedItem := range installedMods {
		for _, portalItem := range detailList {
			if IsBaseMod(installedItem.Name) {
				continue
			}
			if portalItem.Name == installedItem.Name {
				newDL := findModUpgrade(portalItem, installedItem)
				if newDL.Name != "" {
					downloadList = addDownload(newDL, downloadList)
				}
			}
		}
	}

	return downloadList
}

const rd = true

func resolveModDependencies(downloadList []downloadData) ([]downloadData, error) {

	var errStr string
	//Check for unmet dependencies, incompatabilites, etc.
	for _, dl := range downloadList {
		for _, dep := range dl.Data.InfoJSON.Dependencies {
			if rd {
				cwlog.DoLogCW(dl.Name + ": dep: " + dep)
			}

			//Optional dep
			if strings.Contains(dep, "?") {
				if rd {
					cwlog.DoLogCW(dl.Name + ": Skipping optional dep: " + dep)
				}
				continue
			}
			//~ operator not applicable here
			dep = strings.TrimPrefix(dep, "~")
			//Just in case
			dep = strings.TrimSpace(dep)

			depName := dep
			versionNeeded := ""
			equalityOperator := ""
			requiresVersion := false
			parts := strings.Split(dep, " ")
			numParts := len(parts)

			if numParts >= 3 {
				requiresVersion = true
				versionNeeded = parts[numParts-1]
				equalityOperator = parts[numParts-2]
				depName = strings.Join(parts[:numParts-2], " ")
			}

			if IsBaseMod(depName) {
				if rd {
					cwlog.DoLogCW(dl.Name + ": Skipping base mod: " + dep)
				}
				continue
			}

			if strings.Contains(dep, "!") {
				if rd {
					cwlog.DoLogCW(dl.Name + ": checking for incompatible mods: " + dep)
				}
				for m, mod := range downloadList {
					if mod.Name == depName {
						downloadList[m].doDownload = false
						emsg := "Mod " + mod.Name + "-" + mod.Data.Version + " is not compatible with the mod " + dl.Name
						errStr = errStr + emsg
						cwlog.DoLogCW(emsg)
					}
				}
				continue
			}

			//Check locally for deps
			if rd {
				cwlog.DoLogCW(dl.Name + ": Checking locally for deps")
			}
			foundDep := false
			for _, mod := range downloadList {
				//Check if dependency already met
				if mod.Name == depName {
					//If we require a specific version
					if requiresVersion {
						eq := parseOperator(equalityOperator)
						rejectDep, err := checkVersion(eq, versionNeeded, mod.Data.Version)
						if err != nil {
							cwlog.DoLogCW("Unable to parse dependency version:" + dl.Name + ": " + dep)
							continue
						}
						if !rejectDep {
							if rd {
								cwlog.DoLogCW(dl.Name + ": rejecting dep version: " + dep)
							}
							foundDep = true
							break
						}
					} else {
						//Dependency met
						if rd {
							cwlog.DoLogCW(dl.Name + ": Dependency met: " + dep)
						}
						foundDep = true
						break
					}
				}
			}
			if !foundDep {
				if IsBaseMod(depName) || IsBasePrefix(depName) {
					continue
				}
				if rd {
					cwlog.DoLogCW(depName + ": Dependency UNMET")
				}
				newInfo, err := DownloadModInfo(depName)
				if err != nil {
					cwlog.DoLogCW("Unable to download mod info for: " + depName)
					continue
				}
				if rd {
					cwlog.DoLogCW("%v: dep %v info: %v", dl.Name, dep, newInfo.Name)
				}
				vn := "0.0.0"
				if versionNeeded != "" {
					vn = versionNeeded
				}
				newDL := findModUpgrade(newInfo, modZipInfo{Name: depName, Version: vn})
				if rd {
					cwlog.DoLogCW("%v: dep %v upgrade: %v-%v", dl.Name, dep, newDL.Name, newDL.Version)
				}
				newDL.wasDep = true
				if newDL.Name != "" {
					cwlog.DoLogCW("Download added: %v-%v", newDL.Name, newDL.Version)
					downloadList = addDownload(newDL, downloadList)
				}
			}
		}
	}
	if errStr != "" {
		return downloadList, errors.New(errStr)
	} else {
		return downloadList, nil
	}
}

func getDownloadCount(downloadList []downloadData) int {
	count := 0
	for _, dl := range downloadList {
		if dl.doDownload {
			count++
		}
	}
	return count
}

func downloadMods(downloadList []downloadData) string {

	modPath := util.GetModsFolder()

	//Show download status
	downloadCount := getDownloadCount(downloadList)
	if downloadCount > 0 {
		glob.UpdateMessage = nil
		if downloadCount > 1 {
			buf := fmt.Sprintf("Downloading %v mod updates.", downloadCount)
			glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, buf, glob.COLOR_CYAN)
		}
	}
	//Show each download
	var shortBuf string
	errorLog := ""
	for d, dl := range downloadList {
		if !dl.doDownload {
			continue
		}
		if strings.Contains(dl.Name, "!") {
			continue
		}
		if IsBaseMod(dl.Name) {
			continue
		}

		mURL := fmt.Sprintf(displayURL, url.PathEscape(dl.Name))
		longBuf := "[" + dl.Title + "-" + dl.Data.Version + "](" + mURL + ")"

		buf := ""
		if dl.wasDep {
			buf = fmt.Sprintf("Downloading dependency: %v", longBuf)
		} else {
			buf = fmt.Sprintf("Downloading: %v", longBuf)
		}
		cwlog.DoLogCW(buf)

		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, buf, glob.COLOR_CYAN)

		if errorLog != "" {
			glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, dl.Name+": "+errorLog, glob.COLOR_ORANGE)
			errorLog = ""
		}

		//Fetch the mod link
		dlSuffix := fmt.Sprintf(downloadSuffix, cfg.Global.Factorio.Username, cfg.Global.Factorio.Token)
		data, _, err := factUpdater.HttpGet(false, downloadPrefix+dl.Data.DownloadURL+dlSuffix, false)
		if err != nil {
			emsg := "Unable to fetch URL"
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		if !checkSHA1(data, dl.Data.Sha1) {
			emsg := "Mod download is corrupted (invalid hash)."
			cwlog.DoLogCW(emsg)
			errorLog = emsg
			continue
		}

		//Read the mod info.json
		zipIJ := modInfoRead("", data)
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
		}

		downloadList[d].Complete = true

		if downloadCount != 0 {
			shortBuf = shortBuf + ", "
		}
		shortBuf = shortBuf + dl.Name + "-" + dl.Data.Version

		noteMsg := "Installed"
		if dl.OldFilename != "" {
			noteMsg = "Updated"
		}
		newUpdate := ModHistoryItem{
			Name: dl.Name, Notes: noteMsg, Date: time.Now(),
			Version: dl.Data.Version, Filename: dl.Data.FileName,
			OldVersion: dl.OldVersion, OldFilename: dl.OldFilename,
		}
		AddModHistory(newUpdate)
		WriteModHistory()
	}

	return shortBuf
}

// Add mod history, merge "Added by" and "Installed" entries
func AddModHistory(newItem ModHistoryItem) {

	defer WriteModHistory()

	if newItem.Notes == "Installed" {
		for i, item := range ModHistory.History {
			if item.Name == newItem.Name && strings.HasPrefix(item.Notes, "Added by") {
				//Transfer notes
				newItem.Notes = item.Notes
				ModHistory.History[i] = newItem
				return
			}
		}
	}

	for _, item := range ModHistory.History {
		if item.BootItem {
			continue
		}
		if item.Name == newItem.Name {
			if item.Version == newItem.Version {
				//Duplicate entry
				return
			}
		}
	}

	ModHistory.History =
		append(ModHistory.History, newItem)

}

func addDownload(input downloadData, list []downloadData) []downloadData {
	for i, item := range list {
		if item.Name == input.Name {
			//Check versions
			newer, err := checkVersion(EO_GREATER, item.Data.Version, input.Data.Version)
			if err != nil {
				cwlog.DoLogCW("addDownload: Unable to parse version")
				return list
			}
			if newer {
				//Already in list and newer, replace
				list[i] = input
			} else {
				//Already here, but older, skip it
				return list
			}
		}
	}

	return append(list, input)
}

func DownloadModInfo(name string) (modPortalFullData, error) {

	if IsBaseMod(name) {
		return modPortalFullData{}, errors.New("this is a base-game mod")
	}

	name = url.PathEscape(name)

	URL := fmt.Sprintf(modPortalURL, name)
	data, _, err := factUpdater.HttpGet(false, URL, true)
	if err != nil {
		emsg := "Mod info request for " + name + " failed: " + err.Error()
		cwlog.DoLogCW(emsg)
		return modPortalFullData{}, errors.New(emsg)
	}
	newInfo := modPortalFullData{}
	err = json.Unmarshal(data, &newInfo)
	if err != nil {
		emsg := "Mod info unmarshal failed: " + err.Error()
		cwlog.DoLogCW(emsg)
		return modPortalFullData{}, errors.New(emsg)
	}
	return newInfo, nil
}

func findModUpgrade(portalItem modPortalFullData, installedItem modZipInfo) downloadData {

	found := false
	candidateVersion := installedItem.Version
	candidateData := modReleases{}
	for _, release := range portalItem.Releases {
		//Check if this release is newer
		isNewer, err := checkVersion(EO_GREATEREQ, candidateVersion, release.Version)
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

				//Incompatible dependency
				if strings.Contains(dep, "!") {
					continue
				}

				parts := strings.Split(dep, " ")
				numParts := len(parts)
				//Check base mods
				if IsBaseMod(parts[0]) {
					if numParts == 3 {
						eq := parseOperator(parts[1])
						baseReject, err := checkVersion(eq, fact.FactorioVersion, parts[2])
						if baseReject {
							//cwlog.DoLogCW("Rejected: " + installedItem.Name + "-" + release.Version + ". Needs: " + dep)
							reject = true
							continue
						}
						if err != nil {
							cwlog.DoLogCW("Unable to parse version: " + err.Error())
							reject = true
							continue
						}
					}
				}
			}

			//Save this release
			if !reject {
				candidateVersion = release.Version
				candidateData = release
				found = true
			}
		}
	}
	if found {
		//Check if mod is already present in old mods directory before downloading
		//This is great for rolling back to an older version without downloading
		oldMod := util.GetModsFolder() + constants.OldModsDir + "/" + candidateData.FileName
		_, err := os.Stat(oldMod)
		oldModFileNotFound := os.IsNotExist(err)
		if !oldModFileNotFound {
			newMod := util.GetModsFolder() + candidateData.FileName
			err = os.Rename(oldMod, newMod)
			if err != nil {
				cwlog.DoLogCW("Unable to move mod from old mods directory.")
			}
		}

		//Check if mod is already present before downloading
		_, err = os.Stat(util.GetModsFolder() + candidateData.FileName)
		modFileNotFound := os.IsNotExist(err)

		newDL := downloadData{
			Name:  portalItem.Name,
			Title: portalItem.Title,

			Filename:    portalItem.filename,
			OldFilename: installedItem.Filename,

			OldVersion: installedItem.Version,
			Version:    candidateData.Version,

			Data:       candidateData,
			doDownload: modFileNotFound}

		return newDL
	}

	return downloadData{}
}
