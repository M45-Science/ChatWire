package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func MapReset(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Status:", "Resetting map...")

	fact.Map_reset(true)
}
