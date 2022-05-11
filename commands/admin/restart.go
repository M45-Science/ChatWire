package admin

import (
	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboots Factorio only */
func Restart(s *discordgo.Session, i *discordgo.InteractionCreate) {

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
