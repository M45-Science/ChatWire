package admin

import (
	"fmt"
	"strings"

	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

//Set a player's level. Needs to support level names, and have useful help/errors
func SetPlayerLevel(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)

	if argnum > 1 {
		pname := args[0]
		plevelStr := args[1]

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
		}

		//TRUE, modify only
		if fact.PlayerLevelSet(pname, plevel, true) {
			fact.AutoPromote(pname)
			fact.SetPlayerListDirty()
			fact.CMS(m.ChannelID, fmt.Sprintf("Set: Player: %s, Level: %v", pname, fact.LevelToString(plevel)))
			return
		} else {
			fact.CMS(m.ChannelID, fmt.Sprintf("Error: Player not found (case sensitive): %s", pname))
			return
		}
	} else {
		fact.CMS(m.ChannelID, "Invalid level.\nValid levels are:\n`Admin, Regular, Member, New`. Also: `Banned` and `Deleted`.")
	}

}
