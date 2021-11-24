package admin

import (
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/bwmarrin/discordgo"
)

// SaveServer executes the save command on the server.
func SaveServer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	if fact.IsFactRunning() {
		fact.SaveFactorio()
		fact.CMS(m.ChannelID, "Game saved!")
	} else {
		fact.CMS(m.ChannelID, "Factorio isn't running.")
	}
}
