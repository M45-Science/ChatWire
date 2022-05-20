package user

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
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
