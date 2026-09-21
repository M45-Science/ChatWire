package moderator

import (
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"github.com/bwmarrin/discordgo"
)

func PlayerLevel(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	var name, reason string
	level := 0
	for _, arg := range i.ApplicationCommandData().Options {
		switch arg.Name {
		case "name":
			name = arg.StringValue()
		case "level":
			level = int(arg.IntValue())
		case "ban-reason":
			reason = arg.StringValue()
		}
	}
	if err := fact.SetPlayerLevelByModerator(name, level, reason, i.Member.User.Username); err != nil {
		disc.InteractionEphemeralResponse(i, "Error", err.Error())
		return
	}
	disc.InteractionEphemeralResponse(i, "Complete", "Player level updated.")
}
