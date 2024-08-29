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
func PlayerLevel(i *discordgo.InteractionCreate) {

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

		if oldLevel == alevel {
			buf := fmt.Sprintf("Player: %v level was already %v. No action taken.", nplayer.Name, fact.LevelToString(nplayer.Level))
			disc.EphemeralResponse(i, "ERROR:", buf)
			return
		}

		if nplayer != nil {

			/* Unban automatically */
			if alevel >= 0 && oldLevel == -1 {
				fact.WriteFact("/unban " + aname)
			}
			/* Ban automatically */
			if alevel == -1 && oldLevel != -1 {
				reasonString := fmt.Sprintf("%v -- %v %v", reason, banBy, tNow.Format(banTimeFormat))
				fact.WriteFact("/ban " + aname + " " + reasonString)
				nplayer.BanReason = reasonString
			}

			fact.AutoPromote(aname, false, false)
			fact.PlayerLevelSet(nplayer.Name, alevel, true)
			fact.SetPlayerListDirty()
			buf := fmt.Sprintf("Player: %v level set to %v", nplayer.Name, fact.LevelToString(nplayer.Level))
			disc.EphemeralResponse(i, "Complete:", buf)
			return
		}
	}

}
