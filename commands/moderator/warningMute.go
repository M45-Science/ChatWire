package moderator

import (
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/support"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

func WarningPingMute(_ *glob.CommandData, i *discordgo.InteractionCreate) {
	var name string
	minutes := 30
	for _, arg := range i.ApplicationCommandData().Options {
		switch arg.Name {
		case "name":
			name = arg.StringValue()
		case "minutes":
			minutes = int(arg.IntValue())
		}
	}
	until, err := support.SetWarningPingMute(name, minutes)
	if err != nil {
		disc.InteractionEphemeralResponse(i, "Error", err.Error())
		return
	}
	disc.InteractionEphemeralResponse(i, "Complete", fmt.Sprintf("Warning pings for %s are muted until <t:%d:F>. Warnings remain visible.", name, until.Unix()))
}
