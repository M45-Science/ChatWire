package commands

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/commands/admin"
	"ChatWire/glob"
)

func adminCommands() []glob.CommandData {
	return []glob.CommandData{
		{AppCmd: glob.AppCmdData{
			Name:        "config-global",
			Description: "Settings that affect ALL servers.",
			Type:        discordgo.ChatApplicationCommand,
		},
			Function: admin.GConfigServer, AdminOnly: true, PrimaryOnly: true},
	}
}
