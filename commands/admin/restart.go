package admin

import (
	"io"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// Restart saves and restarts the server
func Restart(s *discordgo.Session, m *discordgo.MessageCreate) {

	glob.Refresh = true
	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Now starting!")
	if err != nil {
		support.ErrorLog(err)
	}
	if glob.Running {
		_, err = io.WriteString(glob.Pipe, "/quit\n")
		if err != nil {
			support.ErrorLog(err)
		}
	}
	glob.Shutdown = false

	return
}
