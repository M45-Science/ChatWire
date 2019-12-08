package admin

import (
	"io"
//	"time"
	"github.com/bwmarrin/discordgo"
	"../../support"
	"../../glob"
)

// StopServer saves and stops the server.
func StopServer(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(support.Config.FactorioChannelID,"Server shutting down.")
	io.WriteString(glob.Pipe, "/quit\n")
	glob.Shutdown = true
}
