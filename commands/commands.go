package commands

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/commands/admin"
	"ChatWire/commands/moderator"
	"ChatWire/commands/user"
	"ChatWire/glob"
)

var CL []glob.CommandData

var cmds = []glob.CommandData{

	/* Admin Commands */
	{AppCmd: glob.AppCmdData{
		Name:        "chatwire",
		Description: "Reboot/reload ChatWire.",
		Options: []glob.OptionData{
			{
				Name:        "action",
				Description: "DO NOT use these unless you are certain of what you are doing",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name:     "reboot",
						Value:    "reboot",
						Function: admin.RebootCW,
					},
					{
						Name:     "queue-reboot",
						Value:    "queue-reboot",
						Function: admin.QueReboot,
					},
					{
						Name:     "force-reboot",
						Value:    "force-reboot",
						Function: admin.ForceReboot,
					},
					{
						Name:     "reload-config",
						Value:    "reload-config",
						Function: admin.ReloadConfig,
					},
				},
			},
		},
	},
		AdminOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "map-schedule",
		Description: "Set a map reset schedule.",
		Options: []glob.OptionData{
			{
				Name:        "preset",
				Description: "How often to reset the map.",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name:  "three-months",
						Value: "three months",
					},
					{
						Name:  "two-months",
						Value: "two-months",
					},
					{
						Name:  "monthly",
						Value: "monthly",
					},
					{
						Name:  "twice-monthly",
						Value: "twice-monthly",
					},
					{
						Name:  "day-of-week",
						Value: "day-of-week",
					},
					{
						Name:  "odd-dates",
						Value: "odd-dates",
					},
					{
						Name:  "daily",
						Value: "daily",
					},
					{
						Name:  "no-reset",
						Value: "no-reset",
					},
				},
			},
			{
				Name:        "day",
				Description: "FOR DAY-OF-WEEK PRESET ONLY",
				Type:        discordgo.ApplicationCommandOptionString,
				Choices: []glob.ChoiceData{
					{
						Name:  "monday",
						Value: "MON",
					},
					{
						Name:  "tuesday",
						Value: "TUE",
					},
					{
						Name:  "wednesday",
						Value: "WED",
					},
					{
						Name:  "thursday",
						Value: "THU",
					},
					{
						Name:  "friday",
						Value: "FRI",
					},
					{
						Name:  "saturday",
						Value: "SAT",
					},
					{
						Name:  "sunday",
						Value: "SUN",
					},
					{
						Name:  "default",
						Value: "",
					},
				},
			},
			{
				Name:        "date",
				Description: "For two-week or monthly schedules.",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    0,
				MaxValue:    27,
			},
			{
				Name:        "hour",
				Description: "Hour to reset. (24-hour UTC)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    0,
				MaxValue:    23,
			},
		},
	},
		Function: admin.SetSchedule, ModeratorOnly: true},

	{AppCmd: glob.AppCmdData{
		Name:        "factorio",
		Description: "start, stop, new-map, update, install, etc",
		Options: []glob.OptionData{
			{
				Name:        "action",
				Description: "DO NOT use these unless you are certain of what you are doing!",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name:     "start",
						Value:    "start",
						Function: admin.StartFact,
					},
					{
						Name:     "stop",
						Value:    "stop",
						Function: admin.StopFact,
					},
					{
						Name:     "new-map-preview",
						Value:    "new-map-preview",
						Function: admin.NewMapPreview,
					},
					{
						Name:     "new-map",
						Value:    "new-map",
						Function: admin.NewMap,
					},
					{
						Name:     "update-factorio",
						Value:    "update-factorio",
						Function: admin.UpdateFactorio,
					},
					{
						Name:     "update-mods",
						Value:    "update-mods",
						Function: admin.UpdateMods,
					},
					{
						Name:     "archive-map",
						Value:    "archive-map",
						Function: admin.ArchiveMap,
					},
					{
						Name:     "install-factorio",
						Value:    "install-factorio",
						Function: admin.InstallFact,
					},
				},
			},
		},
	},
		AdminOnly: true},

	{AppCmd: glob.AppCmdData{
		Name:        "config-global",
		Description: "Settings for ALL maps/servers",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: admin.GConfigServer, AdminOnly: true, PrimaryOnly: true},
	/* MODERATOR COMMANDS ---------------- */
	{AppCmd: glob.AppCmdData{
		Name:        "ftp-load",
		Description: "(INCOMPLETE) load a map, mod or modpack from the FTP.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "choose-type",
				Description: "Type of file to load from the FTP.",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name:  "load-map",
						Value: "load-map",
					},
					{
						Name:  "load-mod",
						Value: "load-mod",
					},
					{
						Name:  "load-modpack",
						Value: "load-modpack",
					},
					{
						Name:  "load-settings",
						Value: "load-settings",
					},
				},
			},
		},
	},
		ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "rcon",
		Description: "Remotely run a factorio command",
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
		Name:        "map-reset",
		Description: "Force a map reset, will kick players.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: moderator.MapReset, ModeratorOnly: true},
	{AppCmd: glob.AppCmdData{
		Name:        "config-server",
		Description: "Server settings and options.",
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
				MinValue:    0,
				MaxValue:    23,
			},
			{
				Name:        "end-hour",
				Description: "hour to stop server (24-hour UTC)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    0,
				MaxValue:    23,
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
		Description: "Load a save, lists last 25.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "list",
				Description: "list ALL save files",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "load",
				Description: "specify a save file to load",
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	},
		Function: moderator.ChangeMap, ModeratorOnly: true},

	/* PLAYER COMMMANDS -------------------- */
	{AppCmd: glob.AppCmdData{
		Name:        "list-mods",
		Description: "Show list of mod files",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "list-now",
				Description: "Press ENTER or RETURN to display the list of mods.",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
		Function: moderator.ShowMods},
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
						Name:  "verbose",
						Value: "verbose",
					},
				},
			},
		},
	},
		Function: user.Info},
	{AppCmd: glob.AppCmdData{
		Name:        "modpack",
		Description: "Creates a download link with all mods.",
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
		Description: "REGULARS/VET ONLY: Vote for a new/previous map. Requires TWO vote points.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []glob.OptionData{
			{
				Name:        "vote-now",
				Description: "Press ENTER or RETURN to open the voting box.",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{

				Name:        "moderator",
				Description: "moderator-only options",
				Type:        discordgo.ApplicationCommandOptionString,
				Choices: []glob.ChoiceData{
					{
						Name:  "erase-all",
						Value: "erase-all",
					},
					{
						Name:  "void-all",
						Value: "void-all",
					},
					{
						Name:  "show-all",
						Value: "show-all",
					},
				},
			},
		},
	},
		Function: user.VoteMap},

	{AppCmd: glob.AppCmdData{
		Name:        "pause-game",
		Description: "REGULARS/VET ONLY: Use BEFORE connecting, pauses map while you connect.",
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
}
