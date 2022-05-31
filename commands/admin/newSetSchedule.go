package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
)

func SetSchedule(s *discordgo.Session, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	for _, o := range a.Options {
		arg := o.StringValue()
		err := fact.InterpSchedule(arg, true)
		if err {
			disc.EphemeralResponse(s, i, "Error:", "That is not a valid preset.")
		} else {
			disc.EphemeralResponse(s, i, "Status:", "Schedule set up: "+arg)
			cfg.Local.Options.Schedule = arg
			fact.SetupSchedule()
			cfg.WriteLCfg()
		}
	}
}
