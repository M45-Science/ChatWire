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
			disc.EphemeralResponse(s, i, "Players Online:", "None")
		} else {
			fact.WriteFact("/online")
			fact.CMS(cfg.Local.Channel.ChatChannel, "Players online:")
		}
	} else {
		disc.EphemeralResponse(s, i, "Error:", "Factorio isn't running.")
	}
}
