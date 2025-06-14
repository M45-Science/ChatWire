package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
)

/* Reboots cw */
func ForceReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.InteractionEphemeralResponse(i, "Status:", "Force rebooting ChatWire!")
	glob.RelaunchThrottle = 0
	fact.DoExit(false)
}

/* Reboot when server is empty */
func QueReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.InteractionEphemeralResponse(i, "Complete:", "Chatwire will reboot.")
	glob.RelaunchThrottle = 0
	fact.DoExit(false)
}

/* Reboot when server is empty */
func QueFactReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.InteractionEphemeralResponse(i, "Complete:", "Factorio Reboot has been queued. Server will reboot when map is unoccupied.")
	fact.QueueFactReboot = true
}

/*  Restart saves and restarts the server */
func RebootCW(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.InteractionEphemeralResponse(i, "Status:", "Rebooting ChatWire...")

	disc.InteractionEphemeralResponse(i, "Complete:", "Chatwire will reboot.")
	glob.RelaunchThrottle = 0
	fact.DoExit(false)
}

/* Reload config files */
func ReloadConfig(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	/* Read global and local configs */
	if !cfg.ReadGCfg() {
		buf := "Unable to reload global config file."
		disc.InteractionEphemeralResponse(i, "Error:", buf)
		return
	}
	if !cfg.ReadLCfg() {
		buf := "Unable to reload local config file."
		disc.InteractionEphemeralResponse(i, "Error:", buf)
		return
	}

	/* Re-Write global and local configs */
	cfg.WriteGCfg()
	cfg.WriteLCfg()
	fact.DoUpdateChannelName()
	buf := "Config files have been reloaded."
	disc.InteractionEphemeralResponse(i, "Complete:", buf)

	support.ConfigSoftMod()

	/* This also uses /config to live change settings. */
	fact.GenerateFactorioConfig()

}
