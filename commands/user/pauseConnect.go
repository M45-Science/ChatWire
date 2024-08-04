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
		score := math.Pow(float64(glob.PausedCount+1), 4)
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
				if factname == "" {
					disc.EphemeralResponse(s, i, "Error:", "You need to register to use this command!")
					return
				}
				if fact.IsPlayerOnline(factname) {
					disc.EphemeralResponse(s, i, "Error:", "You are already in the game!")
					return
				}

				/* Otherwise pause game */
				buf := "If you don't attempt to connect within 3 minutes, the pause-on-connect will be canceled.\nIf you don't finish joining the game within 3 minutes, the game will unpause."
				disc.EphemeralResponse(s, i, "Status:", buf)

				glob.PausedForConnect = true
				glob.PausedAt = time.Now()
				glob.PausedConnectAttempt = false
				glob.PausedFor = factname
				glob.PausedCount++

				fact.CMS(cfg.Global.Discord.ReportChannel, cfg.Global.GroupName+"-"+cfg.Local.Callsign+": Pause-on-connect, requested by: "+factname)
			} else {
				buf := fmt.Sprintf("The map was paused %v ago, you must wait an additonal %v to pause again. (%v total)",
					time.Since(glob.PausedAt).Round(time.Second),
					(time.Duration(score)*time.Second)-time.Duration(time.Since(glob.PausedAt).Round(time.Second)),
					time.Duration(score)*time.Second)

				disc.EphemeralResponse(s, i, "Error:", buf)
			}
		} else {
			buf := fmt.Sprintf("A pause-on-connect is already running, requested by: %v", glob.PausedFor)
			disc.EphemeralResponse(s, i, "Error:", buf)
		}
	} else {
		buf := fmt.Sprintf("Sorry, you must have the `%v` role to use this command, see the /register command.", cfg.Global.Discord.Roles.Regular)
		disc.EphemeralResponse(s, i, "Error:", buf)
	}
}
