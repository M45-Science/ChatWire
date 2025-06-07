package commands

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/commands/admin"
	"ChatWire/commands/moderator"
	"ChatWire/commands/user"
	"ChatWire/constants"
	"ChatWire/glob"
)

// Fill in blank choice values from names
func init() {
	for c, command := range cmds {
		for o, option := range command.AppCmd.Options {
			for ch, choice := range option.Choices {
				if choice.Value == nil {
					cmds[c].AppCmd.Options[o].Choices[ch].Value = filterName(choice.Name)
				}
			}
		}
	}
}

var cl []glob.CommandData

var cmds = []glob.CommandData{

	/* Admin Commands */
	{AppCmd: glob.AppCmdData{
		Name:        "chatwire",
		Description: "reboot, or reload config files.",
		Options: []glob.OptionData{
			{
				Name:        "action",
				Description: "DO NOT use these unless you are certain of what you are doing",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name:     "reboot",
						Function: moderator.RebootCW,
					},
					{
						Name:     "queue-reboot",
						Function: moderator.QueReboot,
					},
					{
						Name:     "force-reboot",
						Function: moderator.ForceReboot,
					},
					{
						Name:     "reload-config",
						Function: moderator.ReloadConfig,
					},
					{
						Name:     "queue-fact-reboot",
						Function: moderator.QueFactReboot,
					},
				},
			},
		},
	},
		ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "map-schedule",
		Description: "Change the map reset interval or date.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "interval-months",
				Description: "Set the number of months in the map reset interval",

				Type:     discordgo.ApplicationCommandOptionInteger,
				MinValue: glob.Ptr(0.0),
				MaxValue: glob.Ptr(6.0),
			},
			{
				Name:        "interval-weeks",
				Description: "Set the number of weeks in the map reset interval",

				Type:     discordgo.ApplicationCommandOptionInteger,
				MinValue: glob.Ptr(0.0),
				MaxValue: glob.Ptr(26.0),
			},
			{
				Name:        "interval-days",
				Description: "Set the number of days in the map reset interval",

				Type:     discordgo.ApplicationCommandOptionInteger,
				MinValue: glob.Ptr(0.0),
				MaxValue: glob.Ptr(182.0),
			},
			{
				Name:        "interval-hours",
				Description: "Set the number of hours in the map reset interval",

				Type:     discordgo.ApplicationCommandOptionInteger,
				MinValue: glob.Ptr(0.0),
				MaxValue: glob.Ptr(4320.0),
			},
			{
				Name:        "reset-hour",
				Description: "Force hour to reset, 24 hour format UTC. USE 0 FOR NONE",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    glob.Ptr(0.0),
				MaxValue:    glob.Ptr(23.0),
			},
			{
				Name:        "disable",
				Description: "Turn automatic map resets off",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "reset-date",
				Description: "Temporarily Force Reset-Date (YYYY-MM-DD HH-MM-SS) 24H-UTC",

				Type: discordgo.ApplicationCommandOptionString,
			},
		},
	},
		Function: moderator.SetSchedule, ModeratorOnly: true},

	{AppCmd: glob.AppCmdData{
		Name:        "factorio",
		Description: "Start/stop or update.",
		Options: []glob.OptionData{
			{
				Name:        "action",
				Description: "DO NOT use these unless you are certain of what you are doing!",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name:     "start",
						Function: moderator.StartFact,
					},
					{
						Name:     "stop",
						Function: moderator.StopFact,
					},
					{
						Name:     "new-map",
						Function: moderator.NewMap,
					},
					{
						Name:     "update-mods",
						Function: moderator.UpdateMods,
					},
					{
						Name:     "sync-mods",
						Function: moderator.SyncMods,
					},
					{
						Name:     "archive-map",
						Function: moderator.ArchiveMap,
					},
					{
						Name:     "update-factorio",
						Function: moderator.UpdateFactorio,
					},
					{
						Name:     "install-factorio",
						Function: moderator.InstallFactorio,
					},
				},
			},
		},
	},
		ModeratorOnly: true},

	{AppCmd: glob.AppCmdData{
		Name:        "config-global",
		Description: "Settings that affect ALL servers.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: admin.GConfigServer, AdminOnly: true, PrimaryOnly: true},
	/* MODERATOR COMMANDS ---------------- */
	{AppCmd: glob.AppCmdData{
		Name:        "upload",
		Description: "upload a save-game, mod-list or mod-settings file.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "save-game",
				Description: "select a save-game zip",
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Required:    false,
			},
			{
				Name:        "mod-list",
				Description: "select a " + constants.ModListName + " file",
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Required:    false,
			},
			{
				Name:        "mod-settings",
				Description: "select a " + constants.ModSettingsName + " file",
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Required:    false,
			},
		},
	},
		Function: moderator.UploadFile, ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "editmods",
		Description: "enable/disable or add/remove Factorio mods",
		Type:        discordgo.ChatApplicationCommand,

		Options: []glob.OptionData{
			{
				Name:        "mod-history",
				Description: "Show recent mod history",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "full-history",
				Description: "Show extended mod history (large)",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "clear-history",
				Description: "Clear all mod history and updater blacklist",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "show-mods",
				Description: "List installed mods and version preferences",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "enable-mod",
				Description: "Enable mods by name: mod1, mod2",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "disable-mod",
				Description: "Disable mods by name: mod1, mod2",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "add-mod",
				Description: "Add mods by name or URL. mod1, mod2URL",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "set-version-pref",
				Description: "Set version preference: mod=ver or mod=auto",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "clear-all-mods",
				Description: "Clear entire mod directory (reset)",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
		Function: moderator.EditMods, ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "rcon",
		Description: "Remotely run a factorio command.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "command",
				Description: "factorio command to run",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
		Function: moderator.RCONCmd, ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "rconall",
		Description: "Remotely run a factorio command on all servers.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "command",
				Description: "factorio command to run",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
		Function: moderator.RCONCmd, ModeratorOnly: true, Global: true},
	{AppCmd: glob.AppCmdData{
		Name:        "map-reset",
		Description: "Force a map reset, will kick players.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: moderator.MapReset, ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "config-server",
		Description: "Server settings and options, such as the name.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: moderator.ConfigServer, ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "config-hours",
		Description: "Hours map can be played (24-hour UTC)",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "start-hour",
				Description: "hour to start server (24-hour UTC)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    glob.Ptr(0.0),
				MaxValue:    glob.Ptr(23.0),
			},
			{
				Name:        "end-hour",
				Description: "hour to stop server (24-hour UTC)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    glob.Ptr(0.0),
				MaxValue:    glob.Ptr(23.0),
			},
			{
				Name:        "enabled",
				Description: "hour limits enabled",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
		Function: moderator.ConfigHours, ModeratorOnly: true},

	{AppCmd: glob.AppCmdData{
		Name:        "player-level",
		Description: "Ban, or sets a player's level.",
		Options: []glob.OptionData{
			{
				Name:        "name",
				Description: "Factorio name of player",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "level",
				Description: "player level",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name:  "Moderator",
						Value: 255,
					},
					{
						Name:  "Veteran",
						Value: 3,
					},
					{
						Name:  "Regular",
						Value: 2,
					},
					{
						Name:  "Member",
						Value: 1,
					},
					{
						Name:  "New",
						Value: 0,
					},
					{
						Name:  "Banned",
						Value: -1,
					},
					{
						Name:  "Deleted",
						Value: -255,
					},
				},
			},
			{
				Name:        "ban-reason",
				Description: "reason for ban",
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	},
		Function: moderator.PlayerLevel, ModeratorOnly: true, PrimaryOnly: true},

	{AppCmd: glob.AppCmdData{
		Name:        "change-map",
		Description: "Load a save. Lists last 25 in quick-load drop-down menu if no extra options are chosen.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "list",
				Description: "Print out a list of all save files.",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "load",
				Description: "Manually enter a map name to load.",
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	},
		Function: moderator.ChangeMap, ModeratorOnly: true},

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
				Name:        "pause-now",
				Description: "Pauses map while you connect, expires after 3 mins.",
				Type:        discordgo.ApplicationCommandOptionBoolean,
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
	{AppCmd: glob.AppCmdData{
		Name:        "web-panel",
		Description: "Get a temporary control panel link",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: moderator.WebPanelLink, ModeratorOnly: true},
	/* PLAYER COMMANDS -------------------- */
	{AppCmd: glob.AppCmdData{
		Name:        "list-mods",
		Description: "Show list of mod files",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: user.ListGameMods},
}
