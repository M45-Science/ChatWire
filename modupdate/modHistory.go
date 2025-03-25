package modupdate

import (
	"ChatWire/cfg"
	"fmt"
	"math/rand/v2"
	"strconv"
)

var (
	rollbackList []ModHistoryItem
	rollbackKey  int
)

const (
	keyStart      = 10000
	maxKey        = 99999
	maxModHistory = 250
	maxItemsPage  = 25
)

func ListHistory() string {
	buf := ""

	for i, item := range ModHistory.History {
		if i > maxItemsPage {
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
	if value >= 10000 {
		if int(value) == rollbackKey {
			rollbackKey = 0

			//perform rollback
			cfg.Local.Options.ModUpdate = false
			cfg.WriteLCfg()
			return "(MOCKUP / TEST ONLY -- WIP) Mod roll-back complete!\n**NOTICE: DISABLED AUTOMATIC MOD UPDATER!**"
		}
		return "Invalid roll-back key"
	}

	numHist := uint64(len(ModHistory.History))

	//Unlikely but better to be safe
	if numHist > maxModHistory {
		numHist = maxModHistory
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

		rollbackList = append(rollbackList, item)
		if item.OldFilename != "" {
			buf = buf + "Downgrade " + item.Name + ": " + item.Version + " --> " + item.OldVersion + "\n"
		} else if item.Name != BootName {
			buf = buf + "Remove " + item.Name + "-" + item.Version + "\n"
		}
	}
	if rollbackList != nil {
		rollbackKey = keyStart + rand.IntN(maxKey-keyStart)
		buf = "**Roll-back to #" + strconv.FormatUint(value, 10) + ": ACTION LIST:**\n\n" + buf
		buf = buf + "\n**To perform the roll-back type: `/editmods mod-update-rollback:" + strconv.FormatUint(uint64(rollbackKey), 10) + "`**\n"
	}

	if buf == "" {
		buf = "Unexpected error.\n"
	}
	return buf
}
