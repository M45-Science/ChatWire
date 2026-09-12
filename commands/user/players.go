package user

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

// Players renders the latest structured SoftMod snapshot as Discord embeds.
func Players(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if fact.FactorioBooted && fact.FactIsRunning {

		fact.OnlinePlayersLock.RLock()
		players := append([]glob.OnlinePlayerData(nil), glob.OnlinePlayers...)
		fact.OnlinePlayersLock.RUnlock()
		disc.InteractionEphemeralEmbeds(i, playerListEmbeds(players))
	} else {
		disc.InteractionEphemeralResponse(i, "Error:", "Factorio isn't running.")
	}
}

func playerListEmbeds(players []glob.OnlinePlayerData) []*discordgo.MessageEmbed {
	const fieldsPerEmbed = 25
	server := strings.ToUpper(cfg.Local.Callsign) + "-" + cfg.Local.Name
	if len(players) == 0 {
		return []*discordgo.MessageEmbed{{
			Title:       "Players Online — " + server,
			Description: "No players are currently online.",
			Color:       glob.COLOR_WHITE,
		}}
	}

	limit := min(len(players), fieldsPerEmbed*10)
	embeds := make([]*discordgo.MessageEmbed, 0, (limit+fieldsPerEmbed-1)/fieldsPerEmbed)
	for start := 0; start < limit; start += fieldsPerEmbed {
		end := min(start+fieldsPerEmbed, limit)
		title := fmt.Sprintf("Players Online — %s (%d)", server, len(players))
		if start > 0 {
			title = fmt.Sprintf("Players Online — %s (%d/%d)", server, start+1, len(players))
		}
		embed := &discordgo.MessageEmbed{Title: title, Color: glob.COLOR_GREEN}
		for _, player := range players[start:end] {
			online := (time.Duration(player.TimeTicks) * time.Second / 60).Round(time.Second)
			value := fmt.Sprintf("Score **%.1f h**\nOnline **%s**", float64(player.ScoreTicks)/3600.0, online)
			if player.AFK != "" {
				value += "\nAFK **" + player.AFK + "**"
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   player.Name + " • " + fact.LevelToString(player.Level),
				Value:  value,
				Inline: true,
			})
		}
		embeds = append(embeds, embed)
	}
	if limit < len(players) {
		embeds[len(embeds)-1].Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Showing the first %d players", limit)}
	}
	return embeds
}
