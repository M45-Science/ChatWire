package admin

import (
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

//Archive map
func ShowLocks(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	fact.DoShowLocks(m.ChannelID)
}
