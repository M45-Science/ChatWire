package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func MapReset(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Status:", "Resetting map...")

	//Turn off skip reset flag
	cfg.WriteLCfg()

	fact.Map_reset(true)
}
