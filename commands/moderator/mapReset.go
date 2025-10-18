package moderator

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func MapReset(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Status:", "Resetting map...")

	if err := fact.Map_reset(true); err != nil {
		msg := fmt.Sprintf("Unable to reset map: %v", err)
		disc.InteractionEphemeralResponseColor(i, "Error:", msg, glob.COLOR_RED)
		return
	}
	newHist := modupdate.ModHistoryItem{InfoItem: true,
		Name: "Map Reset By: ", Notes: i.Member.User.Username, Date: time.Now()}
	modupdate.AddModHistory(newHist)
}
