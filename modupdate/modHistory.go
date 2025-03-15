package modupdate

import (
	"fmt"
)

const maxItemsPage = 25

func ListHistory() string {

	buf := ""
	var histCount = 0
	for i, item := range ModHistory.History {
		histCount++

		buf = buf + fmt.Sprintf("%-5v: %v\n",
			i+1, item.Name)
		if item.Notes != "" {
			buf = buf + item.Notes + "\n"
		}
		if item.Version != "" {
			if item.OldVersion != "0.0.0" {
				buf = buf + item.OldVersion + " -> "
			}
			buf = buf + item.Version + "\n"
		}
		buf = buf + item.Date.UTC().Format("01-02-2006 03:04 PM") + " UTC\n\n"

		if histCount >= maxItemsPage {
			break
		}
	}
	if buf == "" {
		buf = "Mod history is empty."
	}

	return buf
}

func ClearHistory() string {
	ModHistory = ModHistoryData{}
	WriteModHistory()
	return "Mod history was cleared."
}
