package user

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func PauseConnect(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	/* regular, mod or admin */
	if !(disc.CheckVeteran(i) || disc.CheckRegular(i) || disc.CheckModerator(i) || disc.CheckAdmin(i)) {
		buf := fmt.Sprintf("Sorry, you must have the `%v` role to use this command, see the /register command.", cfg.Global.Discord.Roles.Regular)
		disc.InteractionEphemeralResponse(i, "Error:", buf)
		return
	}

	action := "pause"
	if len(i.ApplicationCommandData().Options) > 0 {
		option := i.ApplicationCommandData().Options[0]
		if option != nil {
			action = strings.ToLower(strings.TrimSpace(option.StringValue()))
		}
	}

	glob.PausedLock.Lock()
	defer glob.PausedLock.Unlock()

	factname := disc.GetFactorioNameFromDiscordID(i.Member.User.ID)

	/* cancel the existing pause */
	if action == "cancel" {
		if factname == "" {
			disc.InteractionEphemeralResponse(i, "Error:", "You need to register to use this command!")
			return
		}

		if !glob.PausedForConnect {
			disc.InteractionEphemeralResponse(i, "Error:", "No pause-on-connect is currently active.")
			return
		}

		if !strings.EqualFold(glob.PausedFor, factname) && !disc.CheckModerator(i) && !disc.CheckAdmin(i) {
			disc.InteractionEphemeralResponse(i, "Error:", "Only the player who requested the pause or a moderator can cancel it.")
			return
		}

		fact.WriteFact(fmt.Sprintf("/gspeed %0.2f", cfg.Local.Options.Speed))
		glob.PausedForConnect = false
		glob.PausedFor = ""
		glob.PausedConnectAttempt = false
		fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, fmt.Sprintf("Pause-on-connect canceled by: %s", factname))
		disc.InteractionEphemeralResponse(i, "Status:", "Pause-on-connect canceled.")
		return
	}

	if glob.PausedForConnect {
		buf := fmt.Sprintf("A pause-on-connect is already running, requested by: %v", glob.PausedFor)
		disc.InteractionEphemeralResponse(i, "Error:", buf)
		return
	}

	if !fact.FactorioBooted || !fact.FactIsRunning {
		disc.InteractionEphemeralResponse(i, "Error:", "Factorio isn't running right now.")
		return
	}

	/* Throttle */
	score := math.Pow(float64(glob.PausedCount+1), 4)
	if glob.PausedCount == 0 {
		score = 0
	}

	/* If we aren't throttled */
	if time.Since(glob.PausedAt).Seconds() > score {

		/* Check if user is already online */
		if factname == "" {
			disc.InteractionEphemeralResponse(i, "Error:", "You need to register to use this command!")
			return
		}
		if fact.IsPlayerOnline(factname) {
			disc.InteractionEphemeralResponse(i, "Error:", "You are already in the game!")
			return
		}

		/* Otherwise pause game */
		buf := "If you don't attempt to connect within 3 minutes, the pause-on-connect will be canceled.\nIf you don't finish joining the game within 3 minutes, the game will unpause."
		disc.InteractionEphemeralResponse(i, "Status:", buf)

		glob.PausedForConnect = true
		glob.PausedAt = time.Now()
		glob.PausedConnectAttempt = false
		glob.PausedFor = factname
		glob.PausedCount++

		fact.LogGameCMS(true, cfg.Global.Discord.ReportChannel, cfg.Global.GroupName+"-"+cfg.Local.Callsign+": Pause-on-connect, requested by: "+factname)
	} else {
		buf := fmt.Sprintf("The map was paused %v ago, you must wait an additional %v to pause again. (%v total)",
			time.Since(glob.PausedAt).Round(time.Second),
			(time.Duration(score)*time.Second)-time.Duration(time.Since(glob.PausedAt).Round(time.Second)),
			time.Duration(score)*time.Second)

		disc.InteractionEphemeralResponse(i, "Error:", buf)
	}
}
