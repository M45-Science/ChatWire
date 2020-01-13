package admin

import (
	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
	"io"
)

// StopServer saves and stops the server.
func StopServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	if glob.Running {
		glob.Refresh = true
		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Server shutting down.")
		if err != nil {
			support.ErrorLog(err)
		}
		if glob.Pipe != nil && glob.Running {
			_, err = io.WriteString(glob.Pipe, "/quit\n")
			if err != nil {
				support.ErrorLog(err)
			}
		}
		glob.Shutdown = true
	} else {
		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Server isn't running.")
		if err != nil {
			support.ErrorLog(err)
		}
	}

}
