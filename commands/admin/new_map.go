package admin

import (
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Generate map */
func NewMap(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fact.Map_reset("")
}
