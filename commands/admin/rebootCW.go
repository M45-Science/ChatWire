package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/*  Restart saves and restarts the server */
func RebootCW(s *discordgo.Session, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(s, i, "Status:", "Rebooting ChatWire...")

	fact.SetCWReboot(true)
	fact.SetRelaunchThrottle(0)
	fact.QuitFactorio()
}
