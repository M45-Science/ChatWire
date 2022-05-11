package admin

import (
	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboot when server is empty */
func Queue(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsQueued() {
		embed := &discordgo.MessageEmbed{Title: "Complete:", Description: "Reboot has been queued. Server will reboot when map is unoccupied."}
		disc.InteractionResponse(s, i, embed)
		fact.SetQueued(true)
	}
}
