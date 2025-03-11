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

var CL []glob.CommandData

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
		Description: "Change the automatic map reset schedule.",
		Options: []glob.OptionData{
			{
				Name:        "preset",
				Description: "How often to reset the map.",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []glob.ChoiceData{
					{
						Name: "three-months",
					},
					{
						Name: "two-months",
					},
					{
						Name: "monthly",
					},
					{
						Name: "twice-monthly",
					},
					{
						Name: "day-of-week",
					},
					{
						Name: "third-dates",
					},
					{
						Name: "odd-dates",
					},
					{
						Name: "daily",
					},
					{
						Name: "no-reset",
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
				MinValue:    glob.Ptr(float64(0)),
				MaxValue:    glob.Ptr(float64(27)),
			},
			{
				Name:        "hour",
				Description: "Hour to reset. (24-hour UTC)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    glob.Ptr(float64(0)),
				MaxValue:    glob.Ptr(float64(23)),
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
		Description: "enable/disable or add/remove factorio mods.",
		Type:        discordgo.ChatApplicationCommand,

		Options: []glob.OptionData{
			{
				Name:        "mod-history",
				Description: "Display all mod history.",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "clear-history",
				Description: "Clear all mod history.",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "list-mods",
				Description: "List all installed mods",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "enable-mod",
				Description: "Enable a mod by name, use commas to enable more than one mod",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "disable-mod",
				Description: "Disable a mod by name, use commas to disable more than one mod",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "add-mod",
				Description: "Add mods by name or URL, use commas to add more than one mod",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "clear-all-mods",
				Description: "Clear entire mod directory.",
				Type:        discordgo.ApplicationCommandOptionString,
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
				MinValue:    glob.Ptr(float64(0)),
				MaxValue:    glob.Ptr(float64(23)),
			},
			{
				Name:        "end-hour",
				Description: "hour to stop server (24-hour UTC)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				MinValue:    glob.Ptr(float64(0)),
				MaxValue:    glob.Ptr(float64(23)),
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
		Description: "Vote for a new/previous map. Requires TWO vote points.",
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
	/* PLAYER COMMANDS -------------------- */
	{AppCmd: glob.AppCmdData{
		Name:        "list-mods",
		Description: "Show list of mod files",
		Type:        discordgo.ChatApplicationCommand,
	},
		Function: user.ListGameMods},
}
