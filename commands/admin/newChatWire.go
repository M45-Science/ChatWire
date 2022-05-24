package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
)

func ChatWire(s *discordgo.Session, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	for _, o := range a.Options {
		arg := o.StringValue()
		if arg == "reboot" {
			RebootCW(s, i)
			return
		} else if arg == "queue-reboot" {
			QueReboot(s, i)
			return
		} else if arg == "force-reboot" {
			ForceReboot(s, i)
			return
		} else if arg == "reload-config" {
			ReloadConfig(s, i)
			return
		}
	}
}

/* Reboots cw */
func ForceReboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(s, i, "Status:", "Rebooting!")
	glob.RelaunchThrottle = 0
	fact.DoExit(false)
}

/* Reboot when server is empty */
func QueReboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.QueueReload {
		disc.EphemeralResponse(s, i, "Complete:", "Reboot has been queued. Server will reboot when map is unoccupied.")
		fact.QueueReload = true
	}
}

/*  Restart saves and restarts the server */
func RebootCW(s *discordgo.Session, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(s, i, "Status:", "Rebooting ChatWire...")

	glob.DoRebootCW = true
	glob.RelaunchThrottle = 0
	fact.QuitFactorio("Server rebooting...")
}

/* Reload config files */
func ReloadConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {

	/* Read global and local configs */
	if !cfg.ReadGCfg() {
		buf := "Unable to reload global config file."
		disc.EphemeralResponse(s, i, "Error:", buf)
		return
	}
	if !cfg.ReadLCfg() {
		buf := "Unable to reload local config file."
		disc.EphemeralResponse(s, i, "Error:", buf)
		return
	}

	/* Re-Write global and local configs */
	cfg.WriteGCfg()
	cfg.WriteLCfg()
	fact.DoUpdateChannelName(true)
	buf := "Config files have been reloaded."
	disc.EphemeralResponse(s, i, "Complete:", buf)

	/* Config reset-interval */
	if cfg.Local.Options.ScheduleText != "" {
		fact.WriteFact("/resetint " + cfg.Local.Options.ScheduleText)
	}

	if cfg.Local.Options.SoftModOptions.DisableBlueprints {
		fact.WriteFact("/blueprints " + support.BoolToString(!cfg.Local.Options.SoftModOptions.DisableBlueprints))
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Blueprints disabled.")
	}
	if cfg.Local.Options.SoftModOptions.Cheats {
		fact.WriteFact("/enablecheats " + support.BoolToString(cfg.Local.Options.SoftModOptions.Cheats))
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Cheats enabled.")
	}
	/* This also uses /config to live change settings. */
	fact.GenerateFactorioConfig()

}
