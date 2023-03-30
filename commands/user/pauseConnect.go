package user

import (
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"fmt"
	"math"
	"time"

	"github.com/bwmarrin/discordgo"
)

func PauseConnect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if disc.CheckRegular(i) || disc.CheckModerator(i) || disc.CheckAdmin(i) {
		glob.PausedLock.Lock()
		defer glob.PausedLock.Unlock()

		score := 3 * math.Pow(float64(glob.NoResponseCount), 4)

		if !glob.PausedForConnect {
			if time.Since(glob.PausedAt).Seconds() < score {
				buf := "Game paused! If no connection attempts are made within 60 seconds, game will resume."

				var elist []*discordgo.MessageEmbed
				elist = append(elist, &discordgo.MessageEmbed{Title: "Status:", Description: buf})
				f := discordgo.WebhookParams{Embeds: elist}
				disc.FollowupResponse(s, i, &f)

				glob.PausedForConnect = true
				glob.PausedAt = time.Now()
				glob.PausedCount++

				fact.WriteFact("/gspeed 0.02")
			} else {
				buf := fmt.Sprintf("The map was paused %v ago. Sorry, you will have to wait for an additional %v.",
					time.Since(glob.PausedAt),
					time.Duration(time.Since(glob.PausedAt))-(time.Duration(score)*time.Second))

				var elist []*discordgo.MessageEmbed
				elist = append(elist, &discordgo.MessageEmbed{Title: "Status:", Description: buf})
				f := discordgo.WebhookParams{Embeds: elist}
				disc.FollowupResponse(s, i, &f)
			}
		}
	}
}
