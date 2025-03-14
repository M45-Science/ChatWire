package moderator

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
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
		case "reset-date":
			parseResetDate(item.StringValue())
		case "disable":
			n = cfg.ResetInterval{}
		}
	}

	cfg.Local.Options.ResetInterval = n
	fact.SetResetDate()

	if cfg.Local.Options.NextReset.UTC().Sub(time.Now().UTC()) > (time.Hour*24*30*3 + (time.Hour * 24)) {
		disc.InteractionEphemeralResponse(i, "Schedule", "The maximum map reset interval is 3 months, rejecting.")
		cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
		cfg.Local.Options.NextReset = time.Time{}
		return
	}

	cfg.WriteLCfg()
	support.ConfigSoftMod()

	if fact.HasResetInterval() {
		disc.InteractionEphemeralResponse(i, "Schedule", "Schedule set: "+fact.FormatResetInterval()+"\nWill reset at: "+fact.FormatResetTime())
		fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, "❇️ The map reset schedule has been changed: "+fact.FormatResetTime()+" ("+fact.FormatResetInterval()+")")
	} else {
		disc.InteractionEphemeralResponse(i, "Schedule", "Schedule disabled.")
		fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, "❇️ The map reset schedule has been disabled.")
	}
}

func parseResetDate(input string) string {
	layout := "2006-01-02 15-04-05"

	// Parse the time string
	parsedTime, err := time.Parse(layout, input)
	if err != nil {
		return "Unable to parse date provided. Format is 'YYYY-MM-DD HH-MM-SS' (24-hour UTC)"
	}
	cfg.Local.Options.NextReset = parsedTime
	return "Date accepted: " + fact.FormatResetTime() + " (" + fact.TimeTillReset() + ")\n" +
		"Changing the interval will override the new reset date."
}
