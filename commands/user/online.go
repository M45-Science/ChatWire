package user

import (
	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* executes /online on the server, response handled in chat.go */
func PlayersOnline(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if fact.IsFactRunning() {

		if fact.GetNumPlayers() == 0 {
			fact.CMS(cfg.Local.Channel.ChatChannel, "No players are currently online.")
		} else {
			fact.WriteFact("/online")
			fact.CMS(cfg.Local.Channel.ChatChannel, "Players online:")
		}
	} else {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio is currently offline.")
	}
}
