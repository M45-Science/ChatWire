package modupdate

import (
	"fmt"
)

const maxItemsPage = 25

func ListHistory() string {

	bufa := ""
	var histCount = 0
	for i, item := range ModHistory.History {
		histCount++

		bufa = bufa + fmt.Sprintf("%-5v: %v\n",
			i+1, item.Name)
		if item.Notes != "" {
			bufa = bufa + "Notes: " + item.Notes + "\n"
		}
		if item.Version != "" {
			bufa = bufa + "Version: " + item.Version + "\n"
		}
		bufa = bufa + item.Date.UTC().Format("01-02-2006 03:04 PM") + " UTC\n\n"

		if histCount >= maxItemsPage {
			break
		}
	}
	if bufa == "" {
		bufa = "Mod history is empty."
	}
	bufb := ""
	var blackCount = 0
	for i, item := range ModHistory.Blacklist {
		bufb = bufb + fmt.Sprintf("%-5v: %v\n",
			i+1, item.Name)
		if item.Notes != "" {
			bufb = bufb + "Notes: " + item.Notes + "\n"
		}
		if item.Version != "" {
			bufb = bufb + "Version: " + item.Version + "\n"
		}
		bufb = bufb + item.Date.UTC().Format("01-02-2006 03:04 PM") + " UTC\n\n"

		blackCount++
	}
	if bufb == "" {
		bufb = "Updater blacklist is empty."
	}

	buf := bufa + "\n" + bufb + "\n"
	if histCount > 0 {
		buf = buf + "To add an item from the mod history to the updater blacklist, type `/edit-mods mod-history add <number>`\n"
		buf = buf + "Alternatively, you can type the *name** of the mod instead of the item number.\n"
	}
	if blackCount > 0 {
		buf = buf + "To remove an item from the blacklist type `/edit-mods mod-history remove <number>`\n"
	}
	if histCount > maxItemsPage {
		buf = buf + "To see more history, type `/edit-mods mod-history page <number>`\n"
	}
	return buf
}

func ClearHistory() string {
	ModHistory = ModHistoryData{}
	WriteModHistory()
	return "Mod history was cleared."
}
