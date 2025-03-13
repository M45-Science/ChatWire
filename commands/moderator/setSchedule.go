package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func SetSchedule(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	n := cfg.ResetInterval{}
	for _, item := range i.ApplicationCommandData().Options {
		switch item.Name {
		case "Months":
			n.Months = int(item.UintValue())
		case "Weeks":
			n.Weeks = int(item.UintValue())
		case "Days":
			n.Days = int(item.UintValue())
		case "Hours":
			n.Hours = int(item.UintValue())
		}
	}
	cfg.Local.Options.ResetInterval = n
	fact.SetResetDate()
	if fact.HasResetInterval() {
		disc.InteractionEphemeralResponse(i, "Schedule", "Schedule set: "+fact.FormatResetInterval())
	} else {
		disc.InteractionEphemeralResponse(i, "Schedule", "Schedule disabled.")
	}
}
