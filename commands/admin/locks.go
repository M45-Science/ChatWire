package admin

import (
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/bwmarrin/discordgo"
)

//Archive map
func ShowLocks(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	fact.DoShowLocks(m.ChannelID)
}
