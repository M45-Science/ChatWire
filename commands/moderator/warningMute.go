package moderator

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
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
	moderator := "Unknown moderator"
	if i.Member != nil && i.Member.User != nil {
		moderator = i.Member.User.Username
	} else if i.User != nil {
		moderator = i.User.Username
	}
	clean := func(s string) string {
		return strings.ReplaceAll(strings.Join(strings.Fields(s), " "), "@", "@\u200b")
	}
	confirmation := fmt.Sprintf("%s muted warning pings for %s for %d minutes, until <t:%d:F>. Warnings remain visible.", clean(moderator), clean(name), minutes, until.Unix())
	if cfg.Global.Discord.ReportChannel != "" {
		fact.LogCMS(cfg.Global.Discord.ReportChannel, confirmation)
		disc.InteractionEphemeralResponse(i, "Complete", confirmation)
	} else {
		disc.InteractionEphemeralResponse(i, "Complete", confirmation+" No moderation report channel is configured.")
	}
}
