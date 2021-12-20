package admin

import (
	"fmt"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

//Archive map
func ReloadConfig(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	glob.GameMapLock.Lock()
	defer glob.GameMapLock.Unlock()

	//Read global and local configs
	cfg.ReadGCfg()
	cfg.ReadLCfg()

	//Re-Write global and local configs
	cfg.WriteGCfg()
	cfg.WriteLCfg()
	fact.DoUpdateChannelName()
	fact.CMS(m.ChannelID, "Config files reloaded.")

	//Config reset-interval
	if cfg.Local.ResetScheduleText != "" {
		fact.WriteFact("/resetint " + cfg.Local.ResetScheduleText)
	}
	if cfg.Local.DefaultUPSRate > 0 && cfg.Local.DefaultUPSRate < 1000 {
		fact.WriteFact("/aspeed " + fmt.Sprintf("%d", cfg.Local.DefaultUPSRate))
		fact.LogCMS(cfg.Local.ChannelData.ChatID, "Game UPS set to "+fmt.Sprintf("%d", cfg.Local.DefaultUPSRate)+"hz.")
	}
	if cfg.Local.DisableBlueprints {
		fact.WriteFact("/blueprints off")
		fact.LogCMS(cfg.Local.ChannelData.ChatID, "Blueprints disabled.")
	}

}
