package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
)

func MapReset(i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(i, "Status:", "Resetting map...")

	//Turn off skip reset flag
	cfg.Local.Options.SkipReset = false
	cfg.WriteLCfg()

	go fact.Map_reset("", true)
}
