package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Generate map */
func MapReset(s *discordgo.Session, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(s, i, "Status:", "Resetting map...")
	fact.Map_reset("", true)
}
