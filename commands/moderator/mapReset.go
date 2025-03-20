package moderator

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func MapReset(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Status:", "Resetting map...")

	fact.Map_reset(true)
	newHist := modupdate.ModHistoryItem{InfoItem: true,
		Name: "Map Reset By: ", Notes: i.Member.User.Username, Date: time.Now()}
	modupdate.AddModHistory(newHist)
}
