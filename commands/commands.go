package commands

import (
	"fmt"
	"log"

	"ChatWire/cfg"
	"ChatWire/commands/admin"
	"ChatWire/commands/user"
	"ChatWire/disc"

	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Name          string
	Command       func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)
	ModeratorOnly bool
	Help          string
	XHelp         string
	AppCmd        *discordgo.ApplicationCommand
}

var CL []Command

var cmds = []Command{
	/*  Admin Commands */
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "stop-factorio",
		Description: "Stops Factorio, if running.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.StopServer, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "start-factorio",
		Description: "Starts or restarts Factorio, even if already running.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Restart, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "reboot-chatwire",
		Description: "Closes Factorio (if running), and restarts ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Reload, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "force-reboot-chatwire",
		Description: "Big red button. Don't use this lightly. This does not cleanly exit Factorio or ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Reboot, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "queue-reboot",
		Description: "Queues up a reboot. This waits until no players are online to reboot Factorio and ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Queue, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "archive-map",
		Description: "Archives the current map to our website, and posts the link to the chat.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ArchiveMap, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "new-map-preview",
		Description: "Posts a new map, with preview to discord. Use /make-new-map to create it.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.RandomMap, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "make-new-map",
		Description: "Creates a new map and loads it.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Generate, ModeratorOnly: true},

	{Name: "NewMap", Command: admin.NewMap, ModeratorOnly: true,
		Help:  "Map reset",
		XHelp: "Stops Factorio, archives the current map and generates a new one with the current preset."},

	{Name: "UpdateFact", Command: admin.Update, ModeratorOnly: true,
		Help:  "Update Factorio/Cancel",
		XHelp: "Checks if there is there is an update available for Factorio and update if there is. You can: update `CANCEL` to cancel an update."},

	{Name: "PSet", Command: admin.SetPlayerLevel, ModeratorOnly: true,
		Help:  "pset <player> <level>",
		XHelp: "`pset <player> <level>`\nSets the <player> (case sensitive) to <level>\nLevels: `Admin, Regular, Member and New`. Also `Banned` and `Deleted`."},

	{Name: "RCfg", Command: admin.ReloadConfig, ModeratorOnly: true,
		Help:  "Reload configs",
		XHelp: "This reloads the server config files from disk.\nDon't use this, this is only for reloading manually edited config files."},

	{Name: "Set", Command: admin.Set, ModeratorOnly: true,
		Help:  "Set options",
		XHelp: "This allows most options to be set.\nTo see all options, run the command with no options.\nThen: `set <option> <value>`"},

	{Name: "Rewind", Command: admin.Rewind, ModeratorOnly: true,
		Help:  "Rewind map, see autosaves",
		XHelp: "`rewind <autosave number>`\nRunning with no argument shows last 20 autosaves with date stamps. NOTE: Any autosave can be loaded."},

	{Name: "ModUp", Command: admin.ForceUpdateMods, ModeratorOnly: true,
		Help:  "Update installed Factorio mods",
		XHelp: "Forces installed Facorio mods to update, even if automatic mod updaing is disabled."},

	{Name: "SetSeed", Command: admin.SetSeed, ModeratorOnly: true,
		Help:  "Set seed number.",
		XHelp: "Set seed number for next map, 0 for random"},

	{Name: "Debug", Command: admin.DebugStat, ModeratorOnly: true,
		Help:  "debug",
		XHelp: "Development and testing use only."},

	/*  Player Commands */
	{Name: "Whois", Command: user.Whois, ModeratorOnly: false,
		Help:  "Show player info",
		XHelp: "This searches for results (even partial) for the supplied name. The names searched are both `Discord` (if registered) and `Factorio` names.\nOther options: `recent`, `new` and `registered`. These show the most: `recently-online`, `just-joined` and `recently-registered` players. \n`whois <option or name>`"},

	{Name: "Online", Command: user.PlayersOnline, ModeratorOnly: false,
		Help:  "Show players online",
		XHelp: "This just shows players who are currently in the game on this server."},

	{Name: "Info", Command: user.ShowSettings, ModeratorOnly: false,
		Help:  "Server & Map info",
		XHelp: "Shows relevant map/server options and statistics, to see everything type: `info verbose`"},

	{Name: "Register", Command: user.AccessServer, ModeratorOnly: false,
		Help:  "Get an upgraded Discord role!",
		XHelp: "Registration gives you a Discord role that matches your Factorio level. You only need to do this once.\n`Make sure your DMs are turned on in Discord`, or you will not get your registration code. The DM will contain a special code and instructions on how to complete registration.\nThe supplied code is pasted as a `COMMAND in FACTORIO` on the Factorio server with the same name as `the Discord channel` your ran it on. If the code isn't used in a few minutes it will expire. `DO NOT SHARE OR PASTE THIS CODE IN CHAT OR ON DISCORD.`\nIf you `ACCIDENTALLY` paste the code somewhere public, use the `register` command again, to `invalidate the old code` and `receive a new one`.\n"},

	{Name: "Vote-Rewind", Command: user.VoteRewind, ModeratorOnly: false,
		Help:  "Vote to rewind the map",
		XHelp: "This shows the last `20 autosaves` and `all votes`.\nMap-rewind has a one-minute cooldown, and votes expire after `30 minutes`, although you can `change your vote` a few times.\nYou must wait for your old vote to `expire` to vote `again`.\nTo vote: `vote-rewind <autosave number>`\nThis command is only accessible to `REGULARS` on `DISCORD`! (see `help register`).\n`NOTE:` Any autosave can be loaded, not just ones displayed in the command."},
}

func ClearCommands() {
	for _, v := range CL {
		if v.AppCmd != nil {
			disc.DS.ApplicationCommandDelete(disc.DS.State.User.ID, cfg.Global.Discord.Guild, v.AppCmd.ID)
		}
	}
}

/*  RegisterCommands registers the commands on start up. */
func RegisterCommands(s *discordgo.Session) {

	/* Bypasses init loop compile error. */
	CL = append(CL, cmds...)

	for i, c := range CL {
		if c.AppCmd != nil {
			cmd, err := s.ApplicationCommandCreate(cfg.Global.Discord.Application, cfg.Global.Discord.Guild, c.AppCmd)
			if err != nil {
				log.Println("Failed to create command:", c.Name, err)
				continue
			}
			CL[i].AppCmd = cmd
		}
	}
}

func SlashCommand(s *discordgo.Session, i *discordgo.MessageCreate) {
	fmt.Println("MEEP!!!")
}
