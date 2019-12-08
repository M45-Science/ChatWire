package admin

import (
	"io"
	//	"time"
	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// StopServer saves and stops the server.
func StopServer(s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Server shutting down.")
	if err != nil {
		support.ErrorLog(err)
	}
	io.WriteString(glob.Pipe, "/quit\n")
	glob.Shutdown = true
}
