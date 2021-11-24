package admin

import (
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/bwmarrin/discordgo"
)

//Generate map
func NewMap(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	fact.Map_reset("")
}
