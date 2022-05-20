package user

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* executes /online on the server, response handled in chat.go */
func PlayersOnline(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if fact.IsFactRunning() {

		if fact.GetNumPlayers() == 0 {
			disc.EphemeralResponse(s, i, "Players Online:", "None")
		} else {
			buf := ""
			for _, p := range glob.OnlinePlayers {
				buf = buf + fmt.Sprintf("Name: %15v, Score: %5v, Time: %5v, Level: %v\n", p.Name, p.Score, p.Time, p.Level)
			}
			disc.EphemeralResponse(s, i, "Players Online:", buf)
		}
	} else {
		disc.EphemeralResponse(s, i, "Error:", "Factorio isn't running.")
	}
}
