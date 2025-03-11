package modupdate

import (
	"fmt"
)

func ListHistory() string {
	buf := ""
	for i, item := range ModHistory {
		buf = buf + fmt.Sprintf("ID#%03v: Name: %v\nVersion: %10vnDate: %v\n",
			i, item.Name, item.Version, item.Date)
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
