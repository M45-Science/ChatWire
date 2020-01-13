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
		if glob.Pipe != nil && glob.Running {
			io.WriteString(glob.Pipe, "/save\n")
			_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Server saved successfully!")
			if err != nil {
				support.ErrorLog(err)
			}
		}
	}
	return
}
