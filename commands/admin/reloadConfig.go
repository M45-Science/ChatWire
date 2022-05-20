package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/support"
)

/* Reload config files */
func ReloadConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {

	fact.GameMapLock.Lock()
	defer fact.GameMapLock.Unlock()

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
	fact.DoUpdateChannelName()
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
