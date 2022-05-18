package admin

import (
	"fmt"

	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Set a player's level */
func SetPlayerLevel(s *discordgo.Session, i *discordgo.InteractionCreate) {

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
			embed := &discordgo.MessageEmbed{Title: "Complete:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		} else {
			buf := fmt.Sprintf("Player not found: %s", aname)
			embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		}
	}

}
