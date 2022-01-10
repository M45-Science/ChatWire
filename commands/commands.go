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

// Commands is a struct containing a slice of Command.
type Commands struct {
	CommandList []Command
}

// Command is a struct containing fields that hold command information.
type Command struct {
	Name          string
	Command       func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)
	ModeratorOnly bool
	Help          string
	XHelp         string
}

// CL is a Commands interface.
var CL Commands

// RegisterCommands registers the commands on start up.
func RegisterCommands() {
	// Admin Commands
	CL.CommandList = append(CL.CommandList, Command{Name: "Stop", Command: admin.StopServer, ModeratorOnly: true, Help: "Stop Factorio",
		XHelp: "This saves and closes Factorio only."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Start", Command: admin.Restart, ModeratorOnly: true, Help: "Restart Factorio",
		XHelp: "Starts or restarts Factorio only."})
	CL.CommandList = append(CL.CommandList, Command{Name: "RebootCW", Command: admin.Reload, ModeratorOnly: true, Help: "Reboot everything",
		XHelp: "Save and close Facotrio, and reboot ChatWire."})
	CL.CommandList = append(CL.CommandList, Command{Name: "ForceRebootCW", Command: admin.Reboot, ModeratorOnly: true, Help: "Don't use",
		XHelp: "Yeah, this does not cleanly exit, don't use this.\nConsider this the `big red button` that says `do not press`."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Queue", Command: admin.Queue, ModeratorOnly: true, Help: "Queue reboot ",
		XHelp: "Queue a reboot of Factorio and ChatWire.\nRuns once player count is 0."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Save", Command: admin.SaveServer, ModeratorOnly: true, Help: "Force map save",
		XHelp: "This tells Factorio to save the map, thats it..."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Archive", Command: admin.ArchiveMap, ModeratorOnly: true, Help: "Archive current map",
		XHelp: "This takes the current map (if known) and archives it to our website. It also sends the download link to Discord."})
	CL.CommandList = append(CL.CommandList, Command{Name: "RandomMap", Command: admin.RandomMap, ModeratorOnly: true, Help: "Preview new random map",
		XHelp: "If Factorio is shut down, this will generate a preview for a new random map each time. It does not generate the map, use MakeMap to generate the map seen in the last preview."})
	CL.CommandList = append(CL.CommandList, Command{Name: "MakeMap", Command: admin.Generate, ModeratorOnly: true, Help: "Make map previewed",
		XHelp: "If Factorio is shut down, this generates the random map from the last preview."})
	CL.CommandList = append(CL.CommandList, Command{Name: "NewMap", Command: admin.NewMap, ModeratorOnly: true, Help: "Map reset",
		XHelp: "Stops Factorio, archives the map and generates a new one with the current preset."})
	CL.CommandList = append(CL.CommandList, Command{Name: "UpdateFact", Command: admin.Update, ModeratorOnly: true, Help: "Update Factorio/Cancel",
		XHelp: "Checks if there is there is an update available for Factorio and update if there is. You can: update `CANCEL` to cancel an update."})
	CL.CommandList = append(CL.CommandList, Command{Name: "PSet", Command: admin.SetPlayerLevel, ModeratorOnly: true, Help: "Set player level",
		XHelp: "Set player level `-255`=DELETE, `-1`=BANNED, `0`=NEW, `1`=MEMBER, `2`=REGULAR, `255`=Factorio-admin.\n`pset <factorio player name> <level number>`"})
	CL.CommandList = append(CL.CommandList, Command{Name: "RCfg", Command: admin.ReloadConfig, ModeratorOnly: true, Help: "Reload configs",
		XHelp: "This reloads the server config files from disk.\nDon't use this, this is only for reloading manually edited config files."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Set", Command: admin.Set, ModeratorOnly: true, Help: "Set options",
		XHelp: "This allows most options to be set.\nTo see all options, run the command with no options.\nThen: `set <option> <value>`"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Rewind", Command: admin.Rewind, ModeratorOnly: true, Help: "Rewind map, see autosaves",
		XHelp: "`rewind <autosave number>`\nRunning with no argument shows last 40 autosaves with date stamps. NOTE: Any autosave can be loaded."})

	// Util Commands
	CL.CommandList = append(CL.CommandList, Command{Name: "Whois", Command: user.Whois, ModeratorOnly: false, Help: "Show player info",
		XHelp: "This searches for results (even partial) for the supplied name. The names searched are both `Discord` (if registered) and `Factorio` names.\nOther options: `recent`, `new` and `registered`. These show the most: `recently-online`, `just-joined` and `recently-registered` players. \n`whois <option or name>`"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Online", Command: user.PlayersOnline, ModeratorOnly: false, Help: "Show players online",
		XHelp: "This just shows players who are currently in the game on this server."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Info", Command: user.ShowSettings, ModeratorOnly: false, Help: "Server & Map info",
		XHelp: "Shows relevant map/server options and statistics, to see everything use: `info verbose`"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Register", Command: user.AccessServer, ModeratorOnly: false, Help: "Get an upgraded Discord role!",
		XHelp: "Registration gives you a Discord role that matches your Factorio level. You only need to do this once.\n`Make sure your DMs are turned on in Discord`, or you will not get your registration code. The DM will contain a special code and instructions on how to complete registeration.\nThe supplied code is pasted as a `COMMAND in FACTORIO` on the `SAME map/server` as `the Discord channel` your ran it on. If the code isn't used in a few minutes it will expire. `DO NOT SHARE OR PASTE THIS CODE IN CHAT OR ON DISCORD.`\nIf you `ACCIDENTALLY` paste the code somewhere public, use the `register` command again, to `invalidate the old code` and `receive a new one`.\n"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Vote-Rewind", Command: user.VoteRewind, ModeratorOnly: false, Help: "Vote to rewind the map",
		XHelp: "This shows the last `40 autosaves` and `all votes`.\nMap-rewind has a two-minute cooldown, and votes expire after `15 minutes`, although you can `change your vote` a few times.\nYou must wait for your old vote to `expire` to vote `again`.\nTo vote: `vote-rewind <autosave number>`\nThis command is only accessible to `REGULARS` on `DISCORD`! (see `help register`).\n`NOTE:` Any autosave can be loaded, not just ones displayed in the command."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Help", Command: Help, ModeratorOnly: false, Help: "help <command name> for detailed information",
		XHelp: "This command shows help for all commands.\nTo see help for a specific command, use: `help <command name>`.\nIn this case, it is showing additional help for... the help command."})
}

// RunCommand runs a specified command.
func RunCommand(name string, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	for _, command := range CL.CommandList {
		if strings.EqualFold(command.Name, name) {
			if command.ModeratorOnly && disc.CheckModerator(m) {
				command.Command(s, m, args)
			} else if !command.ModeratorOnly {
				command.Command(s, m, args)
			} else {
				fact.CMS(m.ChannelID, "You do not have permission to run this command, smarty pants.")
			}
			return
		}
	}

	fact.CMS(m.ChannelID, "Invalid command, try "+cfg.Global.DiscordCommandPrefix+"help")
}

//Display help, based on user level
func Help(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	buf := ""
	arglen := len(args)
	multiArg := strings.Join(args, " ")
	found := false
	isModerator := disc.CheckModerator(m)
	count := 0

	if arglen > 0 {
		for _, command := range CL.CommandList {
			if !command.ModeratorOnly || (command.ModeratorOnly && isModerator) {
				admin := ""
				if strings.EqualFold(command.Name, args[0]) {
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
				for _, command := range CL.CommandList {
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
		for _, command := range CL.CommandList {
			admin := ""
			if command.ModeratorOnly {
				admin = "(MOD ONLY)"
			}
			buf = buf + fmt.Sprintf("%14v -- %-25v %v\n", cfg.Global.DiscordCommandPrefix+command.Name, command.Help, admin)
		}
	} else {
		for _, command := range CL.CommandList {
			if !command.ModeratorOnly {
				buf = buf + fmt.Sprintf("%14v -- %v\n", cfg.Global.DiscordCommandPrefix+command.Name, command.Help)
			}
		}

	}
	buf = buf + "```"
	fact.CMS(m.ChannelID, buf)
}
