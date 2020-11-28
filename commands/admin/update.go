package admin

import (
	"../../config"
	"../../fact"
	"github.com/bwmarrin/discordgo"
)

func Update(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if config.Config.UpdaterPath != "" {
		fact.CheckFactUpdate(true)
	} else {
		fact.CMS(m.ChannelID, "Updater is not configured.")
	}
}
