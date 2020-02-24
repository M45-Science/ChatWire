package admin

import (
	"fmt"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// StopServer saves and stops the server.
func StatServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	buf := fmt.Sprintf("Version: %s\n", glob.Version)
	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, buf)
	if err != nil {
		support.ErrorLog(err)
	}

}
