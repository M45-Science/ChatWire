package admin

import (
	"io"
	//	"time"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// Restart saves and restarts the server
func Restart(s *discordgo.Session, m *discordgo.MessageCreate) {

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Now restarting!")
	if err != nil {
		support.ErrorLog(err)
	}
	io.WriteString(glob.Pipe, "/quit\n")
	glob.Shutdown = false
	return
}
