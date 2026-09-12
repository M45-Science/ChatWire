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
		playerName := ""
		if nplayer := glob.PlayerList[aname]; nplayer != nil {
			playerName = nplayer.Name
		}
		glob.PlayerListLock.RUnlock()

		if playerName != "" {

			/* Unban automatically */
			if alevel >= 0 && oldLevel == -1 {
				fact.WriteUnban(aname)
			}
			/* Ban automatically */
			if alevel == -1 && oldLevel != -1 {
				reasonString := fmt.Sprintf("%v -- %v %v", reason, banBy, tNow.Format(banTimeFormat))
				fact.WriteBan(aname, reasonString)
				glob.PlayerListLock.Lock()
				if nplayer := glob.PlayerList[aname]; nplayer != nil {
					nplayer.BanReason = reasonString
				}
				glob.PlayerListLock.Unlock()
			}

			fact.PlayerLevelSet(playerName, alevel, true)
			fact.AutoPromoteFromLevel(playerName, false, false, oldLevel)
			// Publish immediately so every ChatWire process can apply the level to
			// its own online copy of the player without waiting for the debounce.
			fact.WritePlayers()
			buf := fmt.Sprintf("Player: %v level set to %v", playerName, fact.LevelToString(alevel))
			disc.InteractionEphemeralResponse(i, "Complete:", buf)
			return
		} else {
			disc.InteractionEphemeralResponse(i, "Error:", "Player not found.")
		}
	} else {
		disc.InteractionEphemeralResponse(i, "Error:", "You must specify a player.")
	}

}
