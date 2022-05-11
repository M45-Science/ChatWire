package user

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* executes /online on the server, response handled in chat.go */
func PlayersOnline(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if fact.IsFactRunning() {

		if fact.GetNumPlayers() == 0 {
			embed := &discordgo.MessageEmbed{Title: "Players Online:", Description: "No players are currently online."}
			disc.InteractionResponse(s, i, embed)
		} else {
			fact.WriteFact("/online")
			fact.CMS(cfg.Local.Channel.ChatChannel, "Players online:")
		}
	} else {
		embed := &discordgo.MessageEmbed{Title: "Notice:", Description: "Factorio is not currently running."}
		disc.InteractionResponse(s, i, embed)
	}
}
