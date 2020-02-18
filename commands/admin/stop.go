package admin

import (
	"io"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// StopServer saves and stops the server.
func StopServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	if glob.Running {
		glob.Refresh = true
		glob.RelaunchThrottle = 0

		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Stopping Factorio...")
		if err != nil {
			support.ErrorLog(err)
		}
		if glob.Running {
			_, err = io.WriteString(glob.Pipe, "/quit\n")
			if err != nil {
				support.ErrorLog(err)
			}
		}
		glob.Shutdown = true
	} else {
		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio isn't running.")
		if err != nil {
			support.ErrorLog(err)
		}
	}

}
