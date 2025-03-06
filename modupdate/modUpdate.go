package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/fact"
)

/* Read entire mod folder */
func CheckMods(force bool, doReport bool) {

	if !cfg.Local.Options.AutoUpdate && !force {
		return
	}

	message := CheckModUpdates(true)
	if message != "" {
		suffix := ""
		if fact.NumPlayers > 0 {
			suffix = " (will upgrade on reboot, when no players are online)"
			fact.FactChat("Mod updates installed: " + message + suffix)
			fact.QueueFactReboot = true
		}
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Mod updates: "+message+suffix)

	}
}
