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
	var gotInterval bool

	buf := ""
	for _, item := range i.ApplicationCommandData().Options {
		switch item.Name {
		case "months":
			n.Months = int(item.UintValue())
			gotInterval = true
		case "weeks":
			n.Weeks = int(item.UintValue())
			gotInterval = true
		case "days":
			n.Days = int(item.UintValue())
			gotInterval = true
		case "hours":
			n.Hours = int(item.UintValue())
			gotInterval = true
		case "reset-hour":
			cfg.Local.Options.ResetHour = int(item.UintValue())
		case "reset-date":
			buf = buf + parseResetDate(item.StringValue())
		case "disable":
			n = cfg.ResetInterval{}
		}
	}

	if gotInterval {
		cfg.Local.Options.ResetInterval = n
		fact.SetResetDate()

		if cfg.Local.Options.NextReset.UTC().Sub(time.Now().UTC()) > (time.Hour*24*30*3 + (time.Hour * 24)) {
			buf = buf + "The maximum map reset interval is 3 months, rejecting."
			disc.InteractionEphemeralResponse(i, "Map Schedule", buf)
			cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
			cfg.Local.Options.NextReset = time.Time{}
			return
		}
	}

	cfg.WriteLCfg()
	support.ConfigSoftMod()

	if fact.HasResetInterval() {
		fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, "❇️ The map reset schedule has been changed:\n"+
			fact.FormatResetTime()+" ("+fact.TimeTillReset()+")\nCurrent reset interval: "+fact.FormatResetInterval())
	} else {
		fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, "❇️ The map reset schedule has been disabled.")
	}

	if buf != "" {
		disc.InteractionEphemeralResponse(i, "Map Schedule", buf)
	} else {
		disc.InteractionEphemeralResponse(i, "Map Schedule", "Accepted")
	}

}

func parseResetDate(input string) string {
	layout := "2006-01-02 15-04-05"

	// Parse the time string
	parsedTime, err := time.Parse(layout, input)
	if err != nil {
		return "Unable to parse date provided. Format is 'YYYY-MM-DD HH-MM-SS' (24-hour UTC)"
	}
	next := cfg.Local.Options.NextReset.UTC().Sub(time.Now().UTC())
	if next < 0 {
		return "That date is in the past, rejecting."
	}

	cfg.Local.Options.NextReset = parsedTime
	return "Date accepted: " + fact.FormatResetTime() + " (" + fact.TimeTillReset() + ")\n" +
		"Changing the interval will override the new reset date."
}
