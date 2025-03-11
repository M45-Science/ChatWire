package modupdate

import (
	"fmt"
)

func ListHistory() string {

	buf := ""
	for i, item := range ModHistory {
		buf = buf + fmt.Sprintf("%-5v: %v\n",
			i+1, item.Name)
		if item.Notes != "" {
			buf = buf + "Notes: \n" + item.Notes
		}
		if item.Version != "" {
			buf = buf + "Version: \n" + item.Version
		}
		buf = buf + item.Date.Format("01-02-2006 03:04 PM")
	}
	if buf == "" {
		buf = "History is empty."
	}
	return buf
}

func ClearHistory() string {
	ModHistory = []ModHistoryData{}
	WriteModHistory()
	return "Mod history was cleared."
}
