package commands

import (
	"fmt"
	"strings"

	"ChatWire/cfg"
	"ChatWire/commands/admin"
	"ChatWire/commands/user"
	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

var CL []Command

var cmds = []Command{
	/*  Admin Commands */
	{Name: "Stop", Command: admin.StopServer, ModeratorOnly: true,
		Help:  "Stop Factorio",
		XHelp: "This saves and closes Factorio only."},

	{Name: "Start", Command: admin.Restart, ModeratorOnly: true,
		Help:  "Restart Factorio",
		XHelp: "Starts or restarts Factorio only."},

	{Name: "RebootCW", Command: admin.Reload, ModeratorOnly: true,
		Help:  "Reboot everything",
		XHelp: "Save and close Factorio, and reboot ChatWire."},

	{Name: "ForceRebootCW", Command: admin.Reboot, ModeratorOnly: true,
		Help:  "Don't use",
		XHelp: "Yeah, this does not cleanly exit, don't use this.\nConsider this the `big red button` that says `do not press`."},

	{Name: "Queue", Command: admin.Queue, ModeratorOnly: true,
		Help:  "Queue reboot ",
		XHelp: "Queue a reboot of Factorio and ChatWire.\nRuns once player count is 0."},

	{Name: "Archive", Command: admin.ArchiveMap, ModeratorOnly: true,
		Help:  "Archive current map",
		XHelp: "This takes the current map (if known) and archives it to our website. It also sends the download link to Discord."},

	{Name: "RandomMap", Command: admin.RandomMap, ModeratorOnly: true,
		Help:  "Previews a new random map",
		XHelp: "If Factorio is shut down, this will generate a preview for a new random map each time. It does not generate the map, use MakeMap to generate the map seen in the last preview."},

	{Name: "MakeMap", Command: admin.Generate, ModeratorOnly: true,
		Help:  "Generates a new map",
		XHelp: "If Factorio is shut down, this generates the random map from the last preview (otherwise a new random map)."},

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

	{Name: "Help", Command: Help, ModeratorOnly: false,
		Help:  "help <command name> for more detailed information",
		XHelp: "This command shows help for all commands.\nTo see help for a specific command, use: `help <command name>`.\nIn this case, it is showing additional help for... the help command."},
}

/*  Commands is a struct containing a slice of Command. */

/*  Command is a struct containing fields that hold command information. */
type Command struct {
	Name          string
	Command       func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)
	ModeratorOnly bool
	Help          string
	XHelp         string
}

/*  RegisterCommands registers the commands on start up. */
func RegisterCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {

	/* Bypasses init loop compile error. */
	CL = append(CL, cmds...)

	for _, _ = range CL {
		s.ApplicationCommandCreate(cfg.Global.DiscordData.AppID, cfg.Global.DiscordData.GuildID, nil)
	}
}

/*  RunCommand runs a specified command. */
func RunCommand(name string, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	for _, command := range CL {
		if strings.EqualFold(command.Name, name) {
			if command.ModeratorOnly && disc.CheckModerator(m) {
				command.Command(s, m, args)
			} else if !command.ModeratorOnly {
				command.Command(s, m, args)
			}
			return
		}
	}

	fact.CMS(m.ChannelID, "Invalid command, try "+cfg.Global.DiscordCommandPrefix+"help")
}

/* Display help, based on player level */
func Help(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	buf := ""
	arglen := len(args)
	multiArg := strings.Join(args, " ")
	found := false
	isModerator := disc.CheckModerator(m)
	count := 0

	if arglen > 0 {
		argOne := strings.TrimPrefix(args[0], cfg.Global.DiscordCommandPrefix)

		for _, command := range CL {
			if !command.ModeratorOnly || (command.ModeratorOnly && isModerator) {
				admin := ""
				if strings.EqualFold(command.Name, argOne) {
					if command.ModeratorOnly {
						admin = " (MOD ONLY)"
					}
					buf = buf + fmt.Sprintf("`%14v -- %-25v %v`\n\n%v", cfg.Global.DiscordCommandPrefix+command.Name, command.Help, admin, command.XHelp)
					count++
					found = true
					fact.CMS(m.ChannelID, buf)
					return
				}
			}
		}
		buf = "`Command not found!`\n\n"
		if len(multiArg) > 2 {
			lMultiArg := strings.ToLower(multiArg)
			buf = buf + "Search results:\n"
			if !found {
				for _, command := range CL {
					if !command.ModeratorOnly || (command.ModeratorOnly && isModerator) {
						lName := strings.ToLower(command.Name)
						lHelp := strings.ToLower(command.Help)
						if strings.Contains(lName, lMultiArg) || strings.Contains(lHelp, lMultiArg) {
							buf = buf + fmt.Sprintf("`%14v -- %v`\n\n%v\n\n", cfg.Global.DiscordCommandPrefix+command.Name, command.Help, command.XHelp)
							count++
						}
					}
				}
			}
		} else {
			buf = buf + "That search term is too broad."
		}
		if count > 0 {
			fact.CMS(m.ChannelID, buf)
		} else {
			fact.CMS(m.ChannelID, "No help was found for that.")
		}
		return
	}

	buf = "```"

	if disc.CheckModerator(m) {
		for _, command := range CL {
			admin := ""
			if command.ModeratorOnly {
				admin = "(MOD ONLY)"
			}
			buf = buf + fmt.Sprintf("%14v -- %-25v %v\n", cfg.Global.DiscordCommandPrefix+command.Name, command.Help, admin)
		}
	} else {
		for _, command := range CL {
			if !command.ModeratorOnly {
				buf = buf + fmt.Sprintf("%14v -- %v\n", cfg.Global.DiscordCommandPrefix+command.Name, command.Help)
			}
		}

	}
	buf = buf + "```"
	fact.CMS(m.ChannelID, buf)
}
