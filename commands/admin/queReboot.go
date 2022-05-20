package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Reboot when server is empty */
func QueReboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsQueued() {
		embed := &discordgo.MessageEmbed{Title: "Complete:", Description: "Reboot has been queued. Server will reboot when map is unoccupied."}
		disc.InteractionResponse(s, i, embed)
		fact.SetQueued(true)
	}
}
