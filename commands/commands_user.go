package commands

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/commands/user"
	"ChatWire/glob"
)

func userCommands() []glob.CommandData {
	return []glob.CommandData{
		/* USER COMMANDS */
		{AppCmd: glob.AppCmdData{
			Name:        "info",
			Description: "Displays status and settings of the server.",
			Type:        discordgo.ChatApplicationCommand,
			Options: []glob.OptionData{
				{
					Name:        "options",
					Description: "verbose shows all settings/info.",
					Type:        discordgo.ApplicationCommandOptionString,
					Choices: []glob.ChoiceData{
						{
							Name: "verbose",
						},
					},
				},
			},
		},
			Function: user.Info},
		{AppCmd: glob.AppCmdData{
			Name:        "modpack",
			Description: "Provides a link to zip file of all game mods.",
			Type:        discordgo.ChatApplicationCommand,
		},
			Function: user.ModPack},

		{AppCmd: glob.AppCmdData{
			Name:        "players",
			Description: "Lists players currently playing.",
			Type:        discordgo.ChatApplicationCommand,
		},
			Function: user.Players},

		{AppCmd: glob.AppCmdData{
			Name:        "vote-map",
			Description: "PRESS ENTER to get a list of maps. Requires TWO vote points.",
			Type:        discordgo.ChatApplicationCommand,
			Options: []glob.OptionData{
				{

					Name:        "moderator",
					Description: "moderator-only options",
					Type:        discordgo.ApplicationCommandOptionString,
					Choices: []glob.ChoiceData{
						{
							Name: "erase-all",
						},
						{
							Name: "void-all",
						},
						{
							Name: "show-all",
						},
					},
				},
			},
		},
			Function: user.VoteMap},

		{AppCmd: glob.AppCmdData{
			Name:        "pause-game",
			Description: "Use BEFORE connecting, pauses map while you connect.",
			Type:        discordgo.ChatApplicationCommand,
			Options: []glob.OptionData{
				{
					Name:        "action",
					Description: "Choose whether to pause for connect or cancel a pending pause.",
					Type:        discordgo.ApplicationCommandOptionString,
					Choices: []glob.ChoiceData{
						{
							Name:  "Pause now",
							Value: "pause",
						},
						{
							Name:  "Cancel pause-on-connect",
							Value: "cancel",
						},
					},
				},
			},
		},
			Function: user.PauseConnect},

		{AppCmd: glob.AppCmdData{
			Name:        "register",
			Description: "Get a discord role, get access to commands.",
			Type:        discordgo.ChatApplicationCommand,
		},
			Function: user.Register, PrimaryOnly: true},
		{AppCmd: glob.AppCmdData{
			Name:        "whois",
			Description: "Get info about <player>",
			Type:        discordgo.ChatApplicationCommand,
			Options: []glob.OptionData{
				{
					Name:        "search",
					Description: "Factorio/Discord name, or any partial match.",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
			Function: user.Whois, PrimaryOnly: true},
		{AppCmd: glob.AppCmdData{
			Name:        "scoreboard",
			Description: "Top 40 players",
			Type:        discordgo.ChatApplicationCommand,
		},
			Function: user.Scoreboard, PrimaryOnly: true},
		/* PLAYER COMMANDS -------------------- */
		{AppCmd: glob.AppCmdData{
			Name:        "list-mods",
			Description: "Show list of mod files",
			Type:        discordgo.ChatApplicationCommand,
		},
			Function: user.ListGameMods},
	}
}
