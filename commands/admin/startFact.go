package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Reboots Factorio only */
func StartFact(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if fact.IsFactorioBooted() {

		buf := "Restarting Factorio..."
		disc.EphemeralResponse(s, i, "Status:", buf)
		fact.QuitFactorio("Server rebooting...")
	} else {
		buf := "Starting Factorio..."
		disc.EphemeralResponse(s, i, "Status:", buf)
	}

	fact.SetAutoStart(true)
	fact.SetRelaunchThrottle(0)
}
