package moderator

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func ConfigHours(i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	buf := ""
	for _, o := range a.Options {
		lbuf := ""
		if strings.EqualFold(o.Name, "enabled") {
			cfg.Local.Options.PlayHourEnable = o.BoolValue()
			lbuf = fmt.Sprintf("hour limits: %v", cfg.Local.Options.PlayHourEnable)
		} else if strings.EqualFold(o.Name, "start-hour") {
			arg := o.IntValue()
			if arg > 0 && arg < 24 {
				cfg.Local.Options.PlayStartHour = int(arg)
				lbuf = fmt.Sprintf("start hour is (GMT): %v", arg)
			}
		} else if strings.EqualFold(o.Name, "end-hour") {
			arg := o.IntValue()
			if arg > 0 && arg < 24 {
				cfg.Local.Options.PlayEndHour = int(arg)
				lbuf = fmt.Sprintf("end hour is (GMT): %v", arg)
			}
		}
		if lbuf != "" {
			buf = buf + lbuf + "\n"
		}
	}
	if buf != "" {
		disc.EphemeralResponse(i, "Status:", buf)
		cfg.WriteLCfg()
	} else {
		disc.EphemeralResponse(i, "Error:", "You didn't supply any options!")
	}
}
