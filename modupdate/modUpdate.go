package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* Read entire mod folder */
func CheckMods(force bool, reportNone bool) {

	if !cfg.Local.Options.AutoUpdate && !force {
		return
	}

	updated, err := CheckModUpdates()
	if reportNone {
		buf := ""
		if err != nil {
			buf = err.Error()
		}
		glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, modUpdateTitle, buf, glob.COLOR_CYAN)
	}
	if updated && err == nil {
		if fact.FactIsRunning {
			fact.QueueFactReboot = true
		}
	}
}
