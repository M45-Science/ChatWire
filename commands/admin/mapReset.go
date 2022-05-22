package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Generate map */
func MapReset(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fact.Map_reset("", true)
	disc.EphemeralResponse(s, i, "Status:", "Resetting map...")
}
