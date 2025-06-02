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
	"sync"
	"time"
)

const modHistoryFile = "modUpdateHistory.dat"

const (
	BootName      = "Factorio Booted"
	InstalledNote = "Installed"
	AddedNote     = "Added by"
	UpdatedNote   = "Updated"
)

func WriteModHistory() {
	ModHistoryLock.Lock()
	defer ModHistoryLock.Unlock()

	tempPath := modHistoryFile + ".tmp"
	finalPath := modHistoryFile

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(ModHistory); err != nil {
		cwlog.DoLogCW("writeModHistory: enc.Encode failure")
		return
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("writeModHistory: os.Create failure")
		return
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("writeModHistory: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("writeModHistory't rename modHistory file.")
		return
	}
}

func ReadModHistory() bool {
	ModHistoryLock.Lock()
	defer ModHistoryLock.Unlock()

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

var modListFileLock sync.Mutex

func WriteModsList(modList ModListData) bool {
	modListFileLock.Lock()
	defer modListFileLock.Unlock()

	finalPath := util.GetModsFolder() + constants.ModListName
	tempPath := finalPath + ".tmp"

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(modList); err != nil {
		cwlog.DoLogCW("writeModsList: enc.Encode failure")
		return false
	}

	os.Mkdir(util.GetModsFolder(), 0755)
	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("writeModsList: os.Create failure")
		return false
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0755)

	if err != nil {
		cwlog.DoLogCW("writeModsList: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("writeModsList: Couldn't rename " + constants.ModListName + ".tmp file.")
		return false
	}

	cwlog.DoLogCW("Wrote " + constants.ModListName)

	return true
}

func getModJSON(data []byte) []byte {
	modListFileLock.Lock()
	defer modListFileLock.Unlock()

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

func operatorToString(input int) string {
	switch input {
	case EO_LESS:
		return "<"
	case EO_LESSEQ:
		return "<="
	case EO_EQUAL:
		return "="
	case EO_GREATEREQ:
		return ">="
	case EO_GREATER:
		return ">"
	default:
		return "ERR"
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
	var modMap map[string]*modZipInfo = map[string]*modZipInfo{}
	for _, mod := range modList {
		if strings.HasSuffix(mod.Name(), ".zip") {
			modInfo := modInfoRead(mod.Name(), nil)
			if modInfo == nil {
				continue
			}
			modInfo.Filename = mod.Name()
			//cwlog.DoLogCW("GetModFiles: " + mod.Name())
			if modMap[modInfo.Name] != nil {
				greater, err := checkVersion(EO_GREATER, modMap[modInfo.Name].Version, modInfo.Version)
				if err == nil && greater {
					cwlog.DoLogCW("Found newer version of mod: %v: %v -> %v", modInfo.Name, modMap[modInfo.Name].Version, modInfo.Version)
					modMap[modInfo.Name] = modInfo
				}
			} else {
				modMap[modInfo.Name] = modInfo
			}
		}
	}

	for _, item := range modMap {
		modFileList = append(modFileList, *item)
	}

	return modFileList, nil
}

func MergeModLists(modFileList []modZipInfo, jsonModList ModListData) []modZipInfo {
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
			installedMods = append(installedMods, modZipInfo{Name: modFile.Name, Enabled: true, Version: modFile.Version})
		}
	}
	for _, mod := range jsonModList.Mods {
		dupe := false
		for i, item := range installedMods {
			if item.Name == mod.Name {
				dupe = true
				installedMods[i].Enabled = mod.Enabled
				break
			}
		}
		if !dupe {
			vers := "0.0.0"
			if IsBaseMod(mod.Name) {
				vers = fact.FactorioVersion
			}
			installedMods = append(installedMods, modZipInfo{Name: mod.Name, Version: vers, Enabled: mod.Enabled})
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

func getDownloadCount(downloadList []downloadData) int {
	return len(downloadList)
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
		//cwlog.DoLogCW("### OldFilename: %v", dl.OldFilename)

		downloadList[d].Complete = true

		if downloadCount != 0 {
			shortBuf = shortBuf + ", "
		}
		shortBuf = shortBuf + dl.Name + "-" + dl.Data.Version

		noteMsg := InstalledNote
		if dl.OldFilename != "" {
			noteMsg = UpdatedNote
		}
		newUpdate := ModHistoryItem{
			Name: dl.Name, Notes: noteMsg, Date: time.Now(),
			Version: dl.Data.Version, Filename: dl.Data.FileName,
			OldVersion: dl.OldVersion, OldFilename: dl.OldFilename,
		}
		AddModHistory(newUpdate)
	}

	return shortBuf
}

// Add mod history, merge "Added by" and "Installed" entries
func AddModHistory(newItem ModHistoryItem) {
	ModHistoryLock.Lock()
	defer func() {
		ModHistoryLock.Unlock()
		WriteModHistory()
	}()

	if newItem.Notes == InstalledNote {
		for i, item := range ModHistory.History {
			if item.Name == newItem.Name && strings.HasPrefix(item.Notes, AddedNote) {
				//Transfer notes
				newItem.Notes = item.Notes
				ModHistory.History[i] = newItem
				return
			}
		}
	}

	if newItem.Name == BootName {
		numItems := len(ModHistory.History) - 1
		if numItems >= 0 && ModHistory.History[numItems].Name == BootName {
			ModHistory.History[numItems] = newItem
			return
		}
	}

	for _, item := range ModHistory.History {
		if item.InfoItem {
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
	modFiles, _ := GetModFiles()

	modList, _ := GetModList()
	mergedMods := MergeModLists(modFiles, modList)

	//Make sure we aren't downloading a mod update we already have
	for _, item := range mergedMods {
		if item.Name == input.Name {
			same, err := checkVersion(EO_EQUAL, item.Version, input.Version)
			if err == nil && same {
				//cwlog.DoLogCW("Skipping mod '%v', mod dupe with older version.", item.Name)
				return list
			}
		}
	}

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
				if resolveDepsDebug {
					cwlog.DoLogCW("Added newer download: %v-%v", input.Name, input.Version)
				}
			} else {
				//Already here, but older, skip it
				if resolveDepsDebug {
					cwlog.DoLogCW("DID NOT ADD download: %v-%v", input.Name, input.Version)
				}
				return list
			}
		}
	}

	if resolveDepsDebug {
		cwlog.DoLogCW("Added download: %v-%v", input.Name, input.Version)
	}
	return append(list, input)
}

var ModInfoLock sync.Mutex

func DownloadModInfo(name string) (modPortalFullData, error) {
	ModInfoLock.Lock()
	defer ModInfoLock.Unlock()

	if IsBaseMod(name) {
		return modPortalFullData{}, nil
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
