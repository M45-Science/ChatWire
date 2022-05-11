package admin

import (
	"fmt"
	"strings"

	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Set a player's level. Needs to support level names, and have useful help/errors */
func SetPlayerLevel(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split("", " ")
	argnum := len(args)

	if argnum > 1 {
		pname := args[0]
		plevelStr := args[1]

		oldLevel := fact.PlayerLevelGet(pname, true)

		plevel := 0
		if strings.EqualFold(plevelStr, "Admin") ||
			strings.EqualFold(plevelStr, "Mod") ||
			strings.EqualFold(plevelStr, "Moderator") {
			plevel = 255
		} else if strings.EqualFold(plevelStr, "Regular") ||
			strings.EqualFold(plevelStr, "Regulars") {
			plevel = 2
		} else if strings.EqualFold(plevelStr, "Member") ||
			strings.EqualFold(plevelStr, "Members") {
			plevel = 1
		} else if strings.EqualFold(plevelStr, "New") ||
			strings.EqualFold(plevelStr, "Clear") ||
			strings.EqualFold(plevelStr, "Reset") {
			plevel = 0
		} else if strings.EqualFold(plevelStr, "Banned") ||
			strings.EqualFold(plevelStr, "Ban") {
			plevel = -1
		} else if strings.EqualFold(plevelStr, "Deleted") ||
			strings.EqualFold(plevelStr, "Delete") {
			plevel = -255
		} else {
			buf := "Invalid level.\nValid levels are:\n`Admin, Regular, Member, New`. Also: `Banned` and `Deleted`."
			embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		}

		/* Unban automatically */
		if plevel >= 0 && oldLevel == -1 {
			fact.WriteFact("/unban " + pname)
		}

		/* TRUE, modify only */
		if fact.PlayerLevelSet(pname, plevel, true) {
			fact.AutoPromote(pname)
			fact.SetPlayerListDirty()
			buf := fmt.Sprintf("Player: %v level set to %v", pname, fact.LevelToString(plevel))
			embed := &discordgo.MessageEmbed{Title: "Complete:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		} else {
			buf := fmt.Sprintf("Player not found: %s", pname)
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
