package moderator

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* Set a player's level */
func PlayerLevel(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	var aname string
	var alevel int
	reason := "No reason given"
	banBy := "Unknown"
	banTimeFormat := "01-02-2006"

	if i.Member != nil {
		banBy = i.Member.User.Username
	}
	tNow := time.Now()

	a := i.ApplicationCommandData()

	//Get args
	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			if arg.Name == "name" {
				aname = strings.ToLower(arg.StringValue())
			} else if arg.Name == "ban-reason" {
				reason = arg.StringValue()
			}
		} else if arg.Type == discordgo.ApplicationCommandOptionInteger {
			alevel = int(arg.IntValue())
		}
	}

	//Only if we have a name
	if aname != "" {

		oldLevel := fact.PlayerLevelGet(aname, false)
		glob.PlayerListLock.RLock()
		nplayer := glob.PlayerList[aname]
		glob.PlayerListLock.RUnlock()

		if nplayer != nil {

			/* Unban automatically */
			if alevel >= 0 && oldLevel == -1 {
				fact.WriteUnban(aname)
			}
			/* Ban automatically */
			if alevel == -1 && oldLevel != -1 {
				reasonString := fmt.Sprintf("%v -- %v %v", reason, banBy, tNow.Format(banTimeFormat))
				fact.WriteBan(aname, reasonString)
				nplayer.BanReason = reasonString
			}

			fact.AutoPromote(aname, false, false)
			fact.PlayerLevelSet(nplayer.Name, alevel, true)
			fact.SetPlayerListDirty()
			buf := fmt.Sprintf("Player: %v level set to %v", nplayer.Name, fact.LevelToString(nplayer.Level))
			disc.InteractionEphemeralResponse(i, "Complete:", buf)
			return
		} else {
			disc.InteractionEphemeralResponse(i, "Error:", "Player not found.")
		}
	} else {
		disc.InteractionEphemeralResponse(i, "Error:", "You must specify a player.")
	}

}
