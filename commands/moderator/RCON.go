package moderator

import (
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"github.com/bwmarrin/discordgo"
)

func RCONCmd(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	var command string
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "command" {
			command = o.StringValue()
		}
	}
	cwlog.DoLogAudit("RCON actor=%s source=discord", i.Member.User.ID)
	out, err := fact.ExecuteRCON(command)
	if err != nil {
		disc.InteractionEphemeralResponse(i, "Error", err.Error())
		return
	}
	if out == "" {
		out = "(Empty response)"
	}
	disc.InteractionEphemeralResponse(i, "Result", out)
}
