package moderator

import (
	"ChatWire/disc"
	"ChatWire/fact"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func ModManager(s *discordgo.Session, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	options := 0
	for _, o := range a.Options {
		options++
		if o.Type == discordgo.ApplicationCommandOptionString {
			arg := o.StringValue()
			if strings.EqualFold(arg, "delete-all") {
				disc.EphemeralResponse(s, i, "Info:", "All mods deleted.")
				return
			} else if strings.EqualFold(arg, "show-list") {
				options = -1
			}
		}
	}
	if options == 0 || options == -1 {
		buf, _ := fact.MakeModList()
		disc.EphemeralResponse(s, i, "Mods available:", "```\n"+buf+"\n```")
	} else {
		disc.EphemeralResponse(s, i, "ERROR:", "Not implemented.")
	}
}
