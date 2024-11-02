package support

import (
	"ChatWire/disc"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

func RunCommandOptions(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	if len(a.Options) > 1 {
		disc.InteractionEphemeralResponse(i, "Error", "Sorry, supply one option only!")
		return
	}
	for _, input := range a.Options {
		arg := input.StringValue()
		for _, option := range cmd.AppCmd.Options {
			for _, choice := range option.Choices {
				if choice.Value == arg {
					if choice.Function == nil {
						continue
					}
					choice.Function(cmd, i)
					return
				}
			}
		}
	}

	disc.InteractionEphemeralResponse(i, "Error", "Sorry, you didn't supply any valid options!")
}
