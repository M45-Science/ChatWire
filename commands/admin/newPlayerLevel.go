package admin

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Set a player's level */
func PlayerLevel(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var aname string
	var alevel int

	a := i.ApplicationCommandData()

	//Get args
	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			aname = arg.StringValue()
		} else if arg.Type == discordgo.ApplicationCommandOptionInteger {
			alevel = int(arg.IntValue())
		}
	}

	//Only if we have a name
	if aname != "" {

		oldLevel := fact.PlayerLevelGet(aname, true)

		/* TRUE, modify only */
		if fact.PlayerLevelSet(aname, alevel, true) {
			/* Unban automatically */
			if alevel >= 0 && oldLevel == -1 {
				fact.WriteFact("/unban " + aname)
			}

			fact.AutoPromote(aname)
			fact.SetPlayerListDirty()
			buf := fmt.Sprintf("Player: %v level set to %v", aname, fact.LevelToString(alevel))
			disc.EphemeralResponse(s, i, "Complete:", buf)
			return
		} else {
			buf := fmt.Sprintf("Player not found: %s", aname)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}
	}

}
