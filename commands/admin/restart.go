package admin

import (
	"io"
//	"time"

	"github.com/bwmarrin/discordgo"
	"../../support"
	"../../glob"
)

// Restart saves and restarts the server
func Restart(s *discordgo.Session, m *discordgo.MessageCreate) {

	s.ChannelMessageSend(support.Config.FactorioChannelID, "Now restarting!")
	io.WriteString(glob.Pipe, "/quit\n")
	glob.Shutdown = false
	return
}
