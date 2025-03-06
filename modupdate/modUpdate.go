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

	sMsg, lMsg, count := CheckModUpdates()
	if count > 0 {
		suffix := ""
		if fact.NumPlayers > 0 {
			suffix = " (will upgrade on reboot, when no players are online)"
			fact.FactChat("Mod updates installed: " + sMsg + suffix)
		}
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "**Mod updates:** "+lMsg+suffix)
		fact.QueueFactReboot = true

	} else if reportNone && count == 0 {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, lMsg)

	}
}
