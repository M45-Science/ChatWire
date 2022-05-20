package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Reboots cw */
func ForceReboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(s, i, "Status:", "Rebooting!")
	fact.SetRelaunchThrottle(0)
	fact.DoExit(false)
}
