package admin

import (
	"../../fact"
	"github.com/bwmarrin/discordgo"
)

//Generate map
func NewMap(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	fact.Map_reset("")
}
