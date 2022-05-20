package admin

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/glob"
)

func DebugStat(s *discordgo.Session, i *discordgo.InteractionCreate) {

	count := 0
	glob.PlayerSusLock.Lock()
	var buf string = "Debug:\nSusList:"
	for pname, score := range glob.PlayerSus {
		if glob.PlayerSus[pname] > 0 {
			count++
			buf = buf + fmt.Sprintf("%v: %v\n", pname, score)
		}
	}

	glob.PlayerSusLock.Unlock()

	glob.ChatterLock.Lock()
	buf = buf + "\nChatterList:"
	for pname, score := range glob.ChatterSpamScore {
		if glob.PlayerSus[pname] > 0 {
			count++
			buf = buf + fmt.Sprintf("%v: %v\n", pname, score)
		}
	}

	glob.ChatterLock.Unlock()

	if count == 0 {
		disc.EphemeralResponse(s, i, "No debug info at this time.", "")
	} else {
		disc.EphemeralResponse(s, i, "Debug Info:", buf)
	}
}
