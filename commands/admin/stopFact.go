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

	if fact.IsFactRunning() {

		buf := "Stopping Factorio."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		fact.QuitFactorio()
	} else {
		buf := "Factorio isn't running, disabling auto-reboot."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
	}

}
