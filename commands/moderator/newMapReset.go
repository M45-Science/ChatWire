package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

func MapReset(s *discordgo.Session, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(s, i, "Status:", "Resetting map...")
	go fact.Map_reset("", true)
}
