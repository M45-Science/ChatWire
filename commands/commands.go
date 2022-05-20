package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/commands/admin"
	"ChatWire/commands/user"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

type Command struct {
	Command       func(s *discordgo.Session, i *discordgo.InteractionCreate)
	ModeratorOnly bool
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
		Command: admin.StopFact, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "start-factorio",
		Description: "Starts OR restarts Factorio, even if already running.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.StartFact, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "reboot-chatwire",
		Description: "Closes Factorio (if running), and restarts ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.RebootCW, ModeratorOnly: true},

	//Add "confirm"
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "force-reboot-chatwire",
		Description: "Big red button. Don't use this lightly. This does not cleanly exit Factorio or ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ForceReboot, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "queue-reboot",
		Description: "Queues up a reboot. This waits until no players are online to reboot Factorio and ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.QueReboot, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "archive-map",
		Description: "Archives the current map to our website, and posts the link to the chat.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ArchiveMap, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "new-map-preview",
		Description: "Posts a new map, with preview to discord. Use /make-new-map after to create it.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.NewMapPrev, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "make-new-map",
		Description: "Creates a new map.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.MakeNewMap, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "map-reset",
		Description: "Stops Factorio, archives current map, generates new one, and starts Factorio.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.MapReset, ModeratorOnly: true},

	//Add the cancel peram
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "update-factorio",
		Description: "Updates Factorio to the latest version if there is a new version available.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "cancel",
				Description: "Cancel an ongoing upgrade, and disable auto-update.",
				Required:    false,
			},
		},
	},

		Command: admin.UpdateFact, ModeratorOnly: true},

	//Complete
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "player-set",
		Description: "Sets a player's rank.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "name",
				Description: "Factorio name of target player",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "level",
				Description: "player level",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Moderator",
						Value: 255,
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
		},
	},
		Command: admin.SetPlayerLevel, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "reload-config",
		Description: "Reloads config files from disk, only used when manually editing config files.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ReloadConfig, ModeratorOnly: true},

	//TODO: Do params
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "config",
		Description: "Change server configuration options.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "name",
				Description: "Factorio server name, excluding callsign.",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "port",
				Description: "UDP Port the game will run on.",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "map-preset",
				Description: "These are built-in Factorio presets",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Default",
						Value: 0,
					},
					{
						Name:  "Rich Resources",
						Value: 1,
					},
					{
						Name:  "Marathon",
						Value: 2,
					},
					{
						Name:  "Death-World",
						Value: 3,
					},
					{
						Name:  "Death-World-Marathon",
						Value: 4,
					},
					{
						Name:  "Rail-World",
						Value: 5,
					},
					{
						Name:  "Ribbon-World",
						Value: 6,
					},
					{
						Name:  "Island",
						Value: 7,
					},
				},
			},
			{
				Name:        "map-generator",
				Description: "A custom map generator",
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "auto-start",
				Description: "If enabled, ChatWire will automatically boot up Factorio.",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "auto-update-factorio",
				Description: "If enabled, ChatWire will automatically update Factorio to the latest stable version.",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Off",
						Value: 0,
					},
					{
						Name:  "Auto-Update (stable)",
						Value: 1,
					},
					{
						Name:  "Auto-Update (experimental)",
						Value: 2,
					},
				},
			},
		},
	},
		Command: admin.Config, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "rewind-map",
		Description: "Rewinds the map to specified autosave.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.RewindMap, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "update-mods",
		Description: "Updates Factorio mods to the latest version if there is a new version available.",
	},
		Command: admin.UpdateMods, ModeratorOnly: true},

	//Add param
	//This should go in config, possibly
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "set-map-seed",
		Description: "Sets the map seed for the next map reset. Value is cleared after use.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.SetSeed, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "debug",
		Description: "Only used for development and testing.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.DebugStat, ModeratorOnly: true},

	/*  Player Commands */
	//Add player
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "whois",
		Description: "Shows information about a player.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.Whois, ModeratorOnly: false},

	//Maybe make this slicker?
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "players-online",
		Description: "Shows detailed info about players currently online.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.PlayersOnline, ModeratorOnly: false},

	//Slicker?
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "server-info",
		Description: "Shows detailed information on the server settings.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.ServerInfo, ModeratorOnly: false},

	//Cleanup, possibly handle other chat channels
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "register",
		Description: "Registers a new account, giving you accociated Discord roles with more privleges.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.Register, ModeratorOnly: false},

	//Add params, make slicker
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "vote-rewind",
		Description: "Vote to rewind the map to the specified autosave (two votes needed!).",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.VoteRewind, ModeratorOnly: false},
}

func ClearCommands() {
	if *glob.DoDeregisterCommands && disc.DS != nil {
		for _, v := range CL {
			if v.AppCmd != nil {
				cwlog.DoLogCW(fmt.Sprintf("Deregistered command: %s", v.AppCmd.Name))
				err := disc.DS.ApplicationCommandDelete(disc.DS.State.User.ID, cfg.Global.Discord.Guild, v.AppCmd.ID)
				if err != nil {
					cwlog.DoLogCW(err.Error())
				}

				time.Sleep(constants.ApplicationCommmandSleep)
			}
		}
	}
}

//https://discord.com/developers/docs/topics/permissions

var modPerms int64 = (1 << 28)    //MANAGE_ROLES
var playerPerms int64 = (1 << 11) //SEND_MESSAGES

/*  RegisterCommands registers the commands on start up. */
func RegisterCommands(s *discordgo.Session) {

	/* Bypasses init loop compile error. */
	CL = append(CL, cmds...)

	//Bypass register, very slow
	//TODO: Cache info and correct for changes when needed

	if *glob.DoRegisterCommands {

		for i, c := range CL {
			if c.AppCmd == nil {
				continue
			}

			if c.ModeratorOnly {
				CL[i].AppCmd.DefaultMemberPermissions = &modPerms
			} else {
				CL[i].AppCmd.DefaultMemberPermissions = &playerPerms
			}

			cmd, err := s.ApplicationCommandCreate(cfg.Global.Discord.Application, cfg.Global.Discord.Guild, c.AppCmd)
			if err != nil {
				log.Println("Failed to create command:", c.AppCmd.Name, err)
				continue
			}
			CL[i].AppCmd = cmd
			cwlog.DoLogCW(fmt.Sprintf("Registered command: %s", c.AppCmd.Name))

			time.Sleep(constants.ApplicationCommmandSleep)

		}
	}

}

func SlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()

	//Don't respond to other channels
	if i.ChannelID == cfg.Local.Channel.ChatChannel && i.AppID == cfg.Global.Discord.Application {
		cwlog.DoLogCW(fmt.Sprintf("%s: command: %s", i.Member.User.Username, data.Name))

		for _, c := range CL {
			if strings.EqualFold(c.AppCmd.Name, data.Name) {
				if c.ModeratorOnly {
					if disc.CheckModerator(i.Member.Roles) {
						c.Command(s, i)
						return
					}
				} else {
					c.Command(s, i)
					return
				}
			}
		}
	}
}
