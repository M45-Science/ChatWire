package admin

import (
	"io"

	"../../glob"
	"../../support"
	"github.com/Distortions81/discordgo"
)

// Restart saves and restarts the server
func Restart(s *discordgo.Session, m *discordgo.MessageCreate) {

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Now restarting!")
	if err != nil {
		support.ErrorLog(err)
	}
	_, err = io.WriteString(glob.Pipe, "/quit\n")
	if err != nil {
		support.ErrorLog(err)
	}
	glob.Shutdown = false
	return
}
