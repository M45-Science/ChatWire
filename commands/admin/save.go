package admin

import (
	"io"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// SaveServer executes the save command on the server.
func SaveServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	if glob.Running {
		glob.Refresh = true
		if glob.Running {
			io.WriteString(glob.Pipe, "/save\n")
			_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Game saved successfully!")
			if err != nil {
				support.ErrorLog(err)
			}
		}
	} else {
		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio isn't running.")
		if err != nil {
			support.ErrorLog(err)
		}
	}
	return
}
