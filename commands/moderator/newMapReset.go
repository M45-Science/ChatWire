package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func MapReset(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(i, "Status:", "Resetting map...")

	//Turn off skip reset flag
	cfg.Local.Options.SkipReset = false
	cfg.WriteLCfg()

	go fact.Map_reset("", true)
}
