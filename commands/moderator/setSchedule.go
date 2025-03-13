package moderator

import (
	"time"

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
	oldDate := cfg.Local.Options.NextReset
	fact.SetResetDate()
	if oldDate.Compare(time.Now().AddDate(0, 3, 1)) > 0 {
		disc.InteractionEphemeralResponse(i, "Schedule", "The maximum map reset interval is 3 months, rejecting.")
		cfg.Local.Options.NextReset = oldDate
		return
	}
	cfg.Local.Options.ResetInterval = n
	cfg.WriteLCfg()

	if fact.HasResetInterval() {
		disc.InteractionEphemeralResponse(i, "Schedule", "Schedule set: "+fact.FormatResetInterval())
	} else {
		disc.InteractionEphemeralResponse(i, "Schedule", "Schedule disabled.")
	}

}
