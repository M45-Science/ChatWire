package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/support"

	"github.com/bwmarrin/discordgo"
)

/* Reload config files */
func ReloadConfig(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	fact.GameMapLock.Lock()
	defer fact.GameMapLock.Unlock()

	/* Read global and local configs */
	if !cfg.ReadGCfg() {
		fact.CMS(m.ChannelID, "Global config file seems to be invalid.")
		return
	}
	if !cfg.ReadLCfg() {
		fact.CMS(m.ChannelID, "Local config file seems to be invalid.")
		return
	}

	/* Re-Write global and local configs */
	cfg.WriteGCfg()
	cfg.WriteLCfg()
	fact.DoUpdateChannelName()
	fact.CMS(m.ChannelID, "Config files reloaded.")

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
