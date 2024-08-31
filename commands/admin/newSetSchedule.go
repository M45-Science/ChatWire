package admin

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func SetSchedule(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	buf := ""
	for _, o := range a.Options {
		lbuf := ""
		if strings.EqualFold(o.Name, "preset") {
			arg := o.StringValue()
			err := fact.InterpSchedule(arg, true)
			fact.UpdateScheduleDesc()
			if err {
				lbuf = "That is not a valid preset."
			} else {
				lbuf = "Schedule set up: " + arg
				cfg.Local.Options.Schedule = arg
			}
		} else if strings.EqualFold(o.Name, "day") {
			arg := o.StringValue()
			if arg != "" {
				cfg.Local.Options.ResetDay = arg
				lbuf = "Day set up: " + arg
			}
		} else if strings.EqualFold(o.Name, "date") {
			arg := o.IntValue()
			if arg > 0 && arg < 29 {
				cfg.Local.Options.ResetDate = int(arg)
				lbuf = fmt.Sprintf("Date set up: %v", arg)
			}
		} else if strings.EqualFold(o.Name, "hour") {
			arg := o.IntValue()
			if arg > 0 && arg < 24 {
				cfg.Local.Options.ResetHour = int(arg)
				lbuf = fmt.Sprintf("Hour set up: %v", arg)
			}
		}
		if lbuf != "" {
			buf = buf + lbuf + "\n"
		}
	}
	if buf != "" {
		disc.EphemeralResponse(i, "Status:", buf)
		fact.SetupSchedule()
		cfg.WriteLCfg()
	}
}
