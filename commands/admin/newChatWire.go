package admin

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
)

func ChatWire(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	for _, o := range a.Options {
		arg := o.StringValue()
		if strings.EqualFold(arg, "reboot") {
			rebootCW(cmd, i)
			return
		} else if strings.EqualFold(arg, "queue-reboot") {
			queReboot(cmd, i)
			return
		} else if strings.EqualFold(arg, "force-reboot") {
			forceReboot(cmd, i)
			return
		} else if strings.EqualFold(arg, "reload-config") {
			reloadConfig(cmd, i)
			return
		}
	}
}

/* Reboots cw */
func forceReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(i, "Status:", "Rebooting!")
	glob.RelaunchThrottle = 0
	fact.DoExit(false)
}

/* Reboot when server is empty */
func queReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(i, "Complete:", "Reboot has been queued. Server will reboot when map is unoccupied.")
	fact.QueueReload = true
}

/*  Restart saves and restarts the server */
func rebootCW(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(i, "Status:", "Rebooting ChatWire...")

	glob.DoRebootCW = true
	glob.RelaunchThrottle = 0
	fact.QuitFactorio("Server rebooting...")
}

/* Reload config files */
func reloadConfig(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

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
