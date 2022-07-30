package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
)

func MapReset(s *discordgo.Session, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(s, i, "Status:", "Resetting map...")

	//Turn off skip reset flag
	cfg.Local.Options.SkipReset = false
	cfg.WriteLCfg()

	go fact.Map_reset("", true)
}
