package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Reboot when server is empty */
func QueReboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsQueued() {
		disc.EphemeralResponse(s, i, "Complete:", "Reboot has been queued. Server will reboot when map is unoccupied.")
		fact.SetQueued(true)
	}
}
