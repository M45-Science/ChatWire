package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/util"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	rollbackList []ModHistoryItem
	rollbackKey  int
)

func ListHistory() string {
	buf := ""

	for i, item := range ModHistory.History {
		if i > constants.ModHistoryPageSize {
			buf = buf + "\n...\n"
			break
		}

		if item.Name == BootName {
			buf = buf + item.Name + "\n"
		} else {
			buf = buf + fmt.Sprintf("**%4v: %v**\n",
				i+1, item.Name)
		}
		if item.Notes != "" {
			buf = buf + item.Notes + "\n"
		}
		if item.Version != "" {
			if item.OldVersion != "0.0.0" {
				buf = buf + item.OldVersion + " -> "
			}
			buf = buf + item.Version + "\n"
		}
		buf = buf + item.Date.UTC().Format("01-02-2006 15:04:05") + " UTC\n\n"
	}
	if buf == "" {
		buf = "Mod history is empty."
	}

	return "**Mod History:**\n\n" + buf
}

func ClearHistory() string {
	ModHistory = ModHistoryData{}
	WriteModHistory()
	return "Mod history was cleared."
}

func ModUpdateRollback(value uint64) string {
	/*
	 * Current plan:
	 * Take input of item number, find it
	 * Make chain of changes to get back
	 * Display list of changes to make, ask for confirm
	 * Stop factorio for changes, with reason note
	 * Apply changes and disable mod update automatically and NOTE UPDATES ARE DISABLED
	 */

	buf := ""
	if value >= constants.ModHistoryKeyStart {
		if int(value) == rollbackKey {
			rollbackKey = 0

			msg := performRollback()
			cfg.Local.Options.ModUpdate = false
			cfg.WriteLCfg()
			return msg + "\n**NOTICE: DISABLED AUTOMATIC MOD UPDATER!**"
		}
		return "Invalid roll-back key"
	}

	numHist := uint64(len(ModHistory.History))

	//Unlikely but better to be safe
	if numHist > constants.MaxModHistory {
		numHist = constants.MaxModHistory
	}

	if value < 1 || value > numHist {
		msg := ListHistory()
		return msg + "\nThat isn't a valid mod history number, please check again."
	}

	//Just in case
	if r := recover(); r != nil {
		return fmt.Sprintf("Panic: Unexpected error: %v", r)
	}

	rollbackList = []ModHistoryItem{}
	for x := value - 1; x < numHist; x++ {
		item := ModHistory.History[x]

		if item.OldFilename == "" {
			oldFile, oldVer := findOlderModFile(item.Name, item.Version)
			if oldFile != "" {
				item.OldFilename = oldFile
				item.OldVersion = oldVer
			}
		}

		rollbackList = append(rollbackList, item)
		if item.OldFilename != "" {
			buf = buf + "Downgrade " + item.Name + ": " + item.Version + " --> " + item.OldVersion + "\n"
		} else if item.Name != BootName {
			buf = buf + "Remove " + item.Name + "-" + item.Version + "\n"
		}
	}
	if rollbackList != nil {
		rollbackKey = constants.ModHistoryKeyStart + rand.IntN(constants.ModHistoryMaxKey-constants.ModHistoryKeyStart)
		buf = "**Roll-back to #" + strconv.FormatUint(value, 10) + ": ACTION LIST:**\n\n" + buf
		buf = buf + "\n**To perform the roll-back type: `/editmods mod-update-rollback:" + strconv.FormatUint(uint64(rollbackKey), 10) + "`**\n"
	}

	if buf == "" {
		buf = "Unexpected error.\n"
	}
	return buf
}

// WriteModHistory persists the global ModHistory to disk.
func WriteModHistory() {
	ModHistoryLock.Lock()
	defer ModHistoryLock.Unlock()

	if err := util.WriteJSONAtomic(modHistoryFile, ModHistory, 0644); err != nil {
		cwlog.DoLogCW("writeModHistory: " + err.Error())
	}
}

// ReadModHistory loads mod history from disk.
func ReadModHistory() bool {
	ModHistoryLock.Lock()
	defer ModHistoryLock.Unlock()

	file, err := os.ReadFile(modHistoryFile)
	if err != nil || file == nil {
		cwlog.DoLogCW("readModHistory: ReadFile failure")
		return false
	}

	newHist := ModHistoryData{}
	if err := json.Unmarshal(file, &newHist); err != nil {
		cwlog.DoLogCW("readModHistory: Unmarshal failure")
		cwlog.DoLogCW(err.Error())
		return false
	}

	ModHistory = newHist
	return true
}

// AddModHistory merges "Added by" and "Installed" entries and saves the update.
func AddModHistory(newItem ModHistoryItem) {
	ModHistoryLock.Lock()
	defer func() {
		ModHistoryLock.Unlock()
		WriteModHistory()
	}()

	if newItem.Notes == InstalledNote {
		for i, item := range ModHistory.History {
			if item.Name == newItem.Name && strings.HasPrefix(item.Notes, AddedNote) {
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
		if item.Name == newItem.Name && item.Version == newItem.Version {
			return
		}
	}

	ModHistory.History = append(ModHistory.History, newItem)
}

func performRollback() string {
	if len(rollbackList) == 0 {
		return "No roll-back data available."
	}

	modPath := cfg.GetModsFolder()
	modList, _ := GetModList()
	actions := ""

	for i := len(rollbackList) - 1; i >= 0; i-- {
		item := rollbackList[i]
		if item.InfoItem || item.Name == BootName {
			continue
		}

		if item.OldFilename == "" {
			oldFile, oldVer := findOlderModFile(item.Name, item.Version)
			if oldFile != "" {
				item.OldFilename = oldFile
				item.OldVersion = oldVer
				rollbackList[i] = item
			}
		}

		if item.OldFilename != "" {
			_ = os.Rename(modPath+item.Filename, modPath+constants.OldModsDir+"/"+item.Filename)
			src := filepath.Join(modPath, constants.OldModsDir, item.OldFilename)
			if _, err := os.Stat(src); err != nil {
				src = filepath.Join(modPath, item.OldFilename)
			}
			_ = os.Rename(src, modPath+item.OldFilename)
			actions = actions + fmt.Sprintf("Downgraded %v to %v\n", item.Name, item.OldVersion)
		} else {
			_ = os.Remove(modPath + item.Filename)
			for m, md := range modList.Mods {
				if md.Name == item.Name {
					modList.Mods = append(modList.Mods[:m], modList.Mods[m+1:]...)
					break
				}
			}
			actions = actions + fmt.Sprintf("Removed %v\n", item.Name)
		}
	}

	WriteModsList(modList)

	cut := len(ModHistory.History) - len(rollbackList)
	if cut < 0 {
		cut = 0
	}
	ModHistory.History = ModHistory.History[:cut]
	WriteModHistory()

	rollbackList = nil

	if actions == "" {
		actions = "No changes applied."
	}
	return "Mod roll-back complete!\n" + actions
}

// findOlderModFile searches the mods and old mods directories for an older
// version of a mod. It returns the filename and version if found.
func findOlderModFile(name, current string) (string, string) {
	modPath := cfg.GetModsFolder()
	dirs := []string{modPath, filepath.Join(modPath, constants.OldModsDir)}
	bestFile := ""
	bestVer := ""

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ver := parseModFileVersion(name, filepath.Join(dir, entry.Name()))
			if ver == "" {
				continue
			}
			if current != "" {
				older, err := checkVersion(EO_LESS, current, ver)
				if err != nil || !older {
					continue
				}
			}
			if bestVer == "" {
				bestVer = ver
				bestFile = entry.Name()
			} else {
				greater, err := checkVersion(EO_GREATER, bestVer, ver)
				if err == nil && greater {
					bestVer = ver
					bestFile = entry.Name()
				}
			}
		}
	}

	return bestFile, bestVer
}

// parseModFileVersion extracts the version from a mod filename.
func parseModFileVersion(name, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	info := modInfoRead("", data)
	if info == nil || info.Name != name {
		return ""
	}
	return info.Version
}
