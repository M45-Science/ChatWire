package user

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* executes /online on the server, response handled in chat.go */
func Players(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if fact.FactorioBooted && fact.FactIsRunning {

		if fact.NumPlayers == 0 {
			disc.EphemeralResponse(s, i, "Players Online:", "None")
		} else {
			buf := ""
			fact.OnlinePlayersLock.Lock()
			for _, p := range glob.OnlinePlayers {
				timeStr := time.Duration(p.TimeTicks * 16666666).Round(time.Second).String()
				buf = buf + fmt.Sprintf("%15v: Score: %5.2v, Time: %6v, Level: %v%v\n", p.Name, p.ScoreTicks/60.0/60.0, timeStr, fact.LevelToString(p.Level), p.AFK)

			}
			fact.OnlinePlayersLock.Unlock()
			disc.EphemeralResponse(s, i, "Players Online:", buf)
		}
	} else {
		disc.EphemeralResponse(s, i, "Error:", "Factorio isn't running.")
	}
}
