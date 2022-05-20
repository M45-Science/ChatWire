package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Reboots Factorio only */
func StartFact(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if fact.IsFactRunning() {

		buf := "Restarting Factorio..."
		embed := &discordgo.MessageEmbed{Title: "Status:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		fact.QuitFactorio()
	} else {
		buf := "Starting Factorio..."
		embed := &discordgo.MessageEmbed{Title: "Status:", Description: buf}
		disc.InteractionResponse(s, i, embed)
	}

	fact.SetAutoStart(true)
	fact.SetRelaunchThrottle(0)
}
