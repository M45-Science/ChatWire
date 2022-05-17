package admin

import (
	"fmt"

	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Set a player's level. Needs to support level names, and have useful help/errors */
func SetPlayerLevel(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var aname string
	var alevel int

	a := i.ApplicationCommandData()
	if !disc.CheckModerator(i.Member.Roles) {
		buf := "This commands is only for moderators."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		return
	}

	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			if arg.StringValue() == "Moderator" {
				aname = arg.StringValue()
			}
		} else if arg.Type == discordgo.ApplicationCommandOptionInteger {
			alevel = int(arg.IntValue())
		}
	}

	if aname != "" {

		oldLevel := fact.PlayerLevelGet(aname, true)

		plevel := 0
		if alevel == 255 {
			plevel = 255
		} else if alevel == 2 {
			plevel = 2
		} else if alevel == 1 {
			plevel = 1
		} else if alevel == 0 {
			plevel = 0
		} else if alevel == -1 {
			plevel = -1
		} else if alevel == -255 {
			plevel = -255
		} else {
			buf := "Invalid level."
			embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		}

		/* Unban automatically */
		if plevel >= 0 && oldLevel == -1 {
			fact.WriteFact("/unban " + aname)
		}

		/* TRUE, modify only */
		if fact.PlayerLevelSet(aname, plevel, true) {
			fact.AutoPromote(aname)
			fact.SetPlayerListDirty()
			buf := fmt.Sprintf("Player: %v level set to %v", aname, fact.LevelToString(plevel))
			embed := &discordgo.MessageEmbed{Title: "Complete:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		} else {
			buf := fmt.Sprintf("Player not found: %s", aname)
			embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		}
	} else {
		buf := "Invalid level.\nValid levels are:\n`Admin, Regular, Member, New`. Also: `Banned` and `Deleted`."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
	}

}
