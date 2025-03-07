package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/fact"
)

/* Read entire mod folder */
func CheckMods(force bool, reportNone bool) {

	if !cfg.Local.Options.AutoUpdate && !force {
		return
	}

	updated, err := CheckModUpdates()
	if updated && err == nil {
		if fact.FactIsRunning {
			fact.QueueFactReboot = true
		}
	}
}
