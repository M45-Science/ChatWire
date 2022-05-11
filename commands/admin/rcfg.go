package admin

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/support"

	"github.com/bwmarrin/discordgo"
)

/* Reload config files */
func ReloadConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {

	fact.GameMapLock.Lock()
	defer fact.GameMapLock.Unlock()

	/* Read global and local configs */
	if !cfg.ReadGCfg() {
		buf := "Unable to reload global config file."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		return
	}
	if !cfg.ReadLCfg() {
		buf := "Unable to reload local config file."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		return
	}

	/* Re-Write global and local configs */
	cfg.WriteGCfg()
	cfg.WriteLCfg()
	fact.DoUpdateChannelName()
	buf := "Config files have been reloaded."
	embed := &discordgo.MessageEmbed{Title: "Complete:", Description: buf}
	disc.InteractionResponse(s, i, embed)

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
