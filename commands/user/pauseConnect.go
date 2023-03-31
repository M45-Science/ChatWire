package user

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"fmt"
	"math"
	"time"

	"github.com/bwmarrin/discordgo"
)

func PauseConnect(s *discordgo.Session, i *discordgo.InteractionCreate) {

	/* regular, mod or admin */
	if disc.CheckRegular(i) || disc.CheckModerator(i) || disc.CheckAdmin(i) {

		/* Lock */
		glob.PausedLock.Lock()
		defer glob.PausedLock.Unlock()

		/* Throttle */
		score := 3 * math.Pow(float64(glob.PausedCount+1), 4)
		if glob.PausedCount == 0 {
			score = 0
		}

		if !fact.FactorioBooted || !fact.FactIsRunning {
			disc.EphemeralResponse(s, i, "Error:", "Factorio isn't running right now.")
			return
		}

		/* Not currently paused */
		if !glob.PausedForConnect {

			/* If we aren't throttled */
			if time.Since(glob.PausedAt).Seconds() > score {

				/* Check if user is already online */
				factname := disc.GetFactorioNameFromDiscordID(i.Member.User.ID)
				if fact.IsPlayerOnline(factname) {
					disc.EphemeralResponse(s, i, "Error:", "You are already in the game!")
					return
				}

				/* Otherwise pause game */
				buf := "If you don't connect within 15 seconds game will unpause.\nIf you don't finish joining the game within 2 minutes, the game will also unpause."
				disc.EphemeralResponse(s, i, "Game Paused:", buf)

				glob.PausedForConnect = true
				glob.PausedAt = time.Now()
				glob.PausedConnectAttempt = false
				glob.PausedFor = factname
				glob.PausedCount++

				fact.WriteFact("/gspeed 0.1")
				fact.CMS(cfg.Local.Channel.ChatChannel, "Pausing game, requested by "+factname)
				fact.CMS(cfg.Global.Discord.ReportChannel, cfg.Global.GroupName+"-"+cfg.Local.Callsign+": Pausing game, requested by: "+factname)
			} else {
				buf := fmt.Sprintf("The map was paused %v ago, you must wait an additonal %v to pause again. (%v total)",
					time.Since(glob.PausedAt).Round(time.Second),
					(time.Duration(score)*time.Second)-time.Duration(time.Since(glob.PausedAt).Round(time.Second)),
					time.Duration(score)*time.Second)

				disc.EphemeralResponse(s, i, "Error:", buf)
			}
		} else {
			buf := fmt.Sprintf("Game is already paused, requested by: %v", glob.PausedFor)
			disc.EphemeralResponse(s, i, "Error:", buf)
		}
	} else {
		buf := fmt.Sprintf("Sorry, you must have the `%v` role to use this command, see the /register command.", cfg.Global.Discord.Roles.Regular)
		disc.EphemeralResponse(s, i, "Error:", buf)
	}
}
