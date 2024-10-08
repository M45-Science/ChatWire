package admin

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

	disc.EphemeralResponse(i, "Status:", "Force rebooting!")
	glob.RelaunchThrottle = 0
	fact.DoExit(false)
}

/* Reboot when server is empty */
func QueReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(i, "Complete:", "Reboot has been queued. Server will reboot when map is unoccupied.")
	fact.QueueReload = true
}

/*  Restart saves and restarts the server */
func RebootCW(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(i, "Status:", "Rebooting ChatWire...")

	glob.DoRebootCW = true
	glob.RelaunchThrottle = 0
	fact.QuitFactorio("Server rebooting...")
}

/* Reload config files */
func ReloadConfig(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	/* Read global and local configs */
	if !cfg.ReadGCfg() {
		buf := "Unable to reload global config file."
		disc.EphemeralResponse(i, "Error:", buf)
		return
	}
	if !cfg.ReadLCfg() {
		buf := "Unable to reload local config file."
		disc.EphemeralResponse(i, "Error:", buf)
		return
	}

	/* Re-Write global and local configs */
	cfg.WriteGCfg()
	cfg.WriteLCfg()
	fact.DoUpdateChannelName()
	buf := "Config files have been reloaded."
	disc.EphemeralResponse(i, "Complete:", buf)

	fact.SetupSchedule()

	support.ConfigSoftMod()

	/* This also uses /config to live change settings. */
	fact.GenerateFactorioConfig()

}
