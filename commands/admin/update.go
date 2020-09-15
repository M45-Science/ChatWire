package admin

import (
	"../../fact"
	"github.com/bwmarrin/discordgo"
)

func Update(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	fact.CheckFactUpdate(true)
}
