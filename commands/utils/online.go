package utils

import (
	//"fmt"

	"io"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
	//b64 "encoding/base64"
)

// online executes /p o on the server.
func PlayersOnline(s *discordgo.Session, m *discordgo.MessageCreate) {

	if glob.Running {
		glob.Refresh = true

		if glob.Running {
			io.WriteString(glob.Pipe, "/players online\n")
			_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Players online:")
			if err != nil {
				support.ErrorLog(err)
			}
		}
	} else {
		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio is currently offline.")
		if err != nil {
			support.ErrorLog(err)
		}
	}
	return
}
