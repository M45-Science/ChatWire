package moderator

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* Set a player's level */
func PlayerLevel(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var aname string
	var alevel int

	a := i.ApplicationCommandData()

	//Get args
	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			aname = strings.ToLower(arg.StringValue())
		} else if arg.Type == discordgo.ApplicationCommandOptionInteger {
			alevel = int(arg.IntValue())
		}
	}

	//Only if we have a name
	if aname != "" {

		oldLevel := fact.PlayerLevelGet(aname, true)
		nplayer := glob.PlayerList[aname]

		if nplayer != nil {

			/* TRUE, modify only */
			if oldLevel == alevel {
				buf := fmt.Sprintf("Player: %v: level is already the same as requested: %v, no action taken.", nplayer.Name, fact.LevelToString(nplayer.Level))
				disc.EphemeralResponse(s, i, "Error:", buf)
			} else {
				/* Unban automatically */
				if alevel >= 0 && oldLevel == -1 {
					fact.WriteFact("/unban " + aname)
				}
				/* Ban automatically */
				if alevel == -1 && oldLevel != -1 {
					fact.WriteFact("/ban " + aname)
				}

				fact.AutoPromote(aname)
				fact.PlayerLevelSet(nplayer.Name, alevel, true)
				fact.SetPlayerListDirty()
				buf := fmt.Sprintf("Player: %v level set to %v", nplayer.Name, fact.LevelToString(nplayer.Level))
				disc.EphemeralResponse(s, i, "Complete:", buf)
				return
			}
		} else {
			buf := fmt.Sprintf("Player not found: %v", aname)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}
	}

}
