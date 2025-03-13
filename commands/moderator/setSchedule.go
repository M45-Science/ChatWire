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
		case "months":
			n.Months = int(item.UintValue())
		case "weeks":
			n.Weeks = int(item.UintValue())
		case "days":
			n.Days = int(item.UintValue())
		case "hours":
			n.Hours = int(item.UintValue())
		case "reset-hour":
			cfg.Local.Options.ResetHour = int(item.UintValue())
		case "disable":
			n = cfg.ResetInterval{}
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
