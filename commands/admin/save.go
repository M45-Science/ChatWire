package admin

import (
	"io"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

// SaveServer executes the save command on the server.
func SaveServer(s *discordgo.Session, m *discordgo.MessageCreate) {
	io.WriteString(glob.Pipe, "/save\n")
	s.ChannelMessageSend(support.Config.FactorioChannelID, "Server saved successfully!")
	return
}
