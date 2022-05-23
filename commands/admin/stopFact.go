package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/*  StopServer saves the map and closes Factorio.  */
func StopFact(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fact.SetRelaunchThrottle(0)
	fact.SetAutoStart(false)

	if fact.IsFactorioBooted() {

		buf := "Stopping Factorio."
		disc.EphemeralResponse(s, i, "Status:", buf)
		fact.QuitFactorio("Server quitting...")
	} else {
		buf := "Factorio isn't running, disabling auto-reboot."
		disc.EphemeralResponse(s, i, "Warning:", buf)
	}

}
