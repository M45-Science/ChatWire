package moderator

import (
	"ChatWire/disc"
	"ChatWire/firewall"
	"ChatWire/glob"
	"github.com/bwmarrin/discordgo"
)

func IPBan(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	var action, ip string
	for _, arg := range i.ApplicationCommandData().Options {
		switch arg.Name {
		case "action":
			action = arg.StringValue()
		case "ip":
			ip = arg.StringValue()
		}
	}
	out, e := firewall.Execute(action, ip, i.Member.User.ID)
	if e != nil {
		disc.InteractionEphemeralResponse(i, "Error", e.Error())
		return
	}
	if out == "" {
		out = "No matching deny rules."
	}
	disc.InteractionEphemeralResponse(i, "Firewall", out)
}
