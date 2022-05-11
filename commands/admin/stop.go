package admin

import (
	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/*  StopServer saves the map and closes Factorio.  */
func StopServer(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
