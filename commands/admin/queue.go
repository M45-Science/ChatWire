package admin

import (
	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// Restart saves and restarts the server
func Queue(s *discordgo.Session, m *discordgo.MessageCreate) {

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "reload is now queued.")
	if err != nil {
		support.ErrorLog(err)
	}
	glob.QueueReload = true
	return
}
