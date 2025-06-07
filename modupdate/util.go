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
	path := cfg.GetModsFolder() + constants.ModListName

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
		path := cfg.GetModsFolder() + "/" + modName

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

	finalPath := cfg.GetModsFolder() + constants.ModListName
	os.Mkdir(cfg.GetModsFolder(), 0755)

	if err := util.WriteJSONAtomic(finalPath, modList, 0755); err != nil {
		cwlog.DoLogCW("writeModsList: " + err.Error())
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
	modList, err := os.ReadDir(cfg.GetModsFolder())
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

func getFactorioVersion() {
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

	modPath := cfg.GetModsFolder()

	//Show download status
	downloadCount := getDownloadCount(downloadList)
	if downloadCount > 0 {
		glob.ResetUpdateMessage()
		if downloadCount > 1 {
			buf := fmt.Sprintf("Downloading %v mod updates.", downloadCount)
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), modUpdateTitle, buf, glob.COLOR_CYAN))
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

		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), modUpdateTitle, buf, glob.COLOR_CYAN))

		if errorLog != "" {
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), modUpdateTitle, dl.Name+": "+errorLog, glob.COLOR_ORANGE))
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

		//Delete old mod file, if we had one
		if dl.OldFilename != "" {
			err = os.Remove(modPath + dl.OldFilename)
			if err != nil && !os.IsNotExist(err) {
				emsg := "Unable to remove old mod file"
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
