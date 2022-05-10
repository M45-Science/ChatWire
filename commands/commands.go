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
	Command       func(s *discordgo.Session, i *discordgo.InteractionCreate)
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
		Description: "Posts a new map, with preview to discord. Use /make-new-map after to create it.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.RandomMap, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "make-new-map",
		Description: "Creates a new map.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Generate, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "map-reset",
		Description: "Stops Factorio, archives current map, generates new one, and starts Factorio.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.NewMap, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "update-factorio",
		Description: "Updates Factorio to the latest version if there is a new version available.",
		Type:        discordgo.ChatApplicationCommand,
	},

		Command: admin.Update, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "player-set",
		Description: "Sets a player's rank.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.SetPlayerLevel, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "reload-config",
		Description: "Reloads config files from disk, only used when manually editing config files.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ReloadConfig, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "config",
		Description: "Change server configuration options.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Set, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "rewind-map",
		Description: "Rewinds the map to specified autosave.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Rewind, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "update-mods",
		Description: "Updates Factorio mods to the latest version if there is a new version available.",
	},
		Command: admin.ForceUpdateMods, ModeratorOnly: true},

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
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "whois",
		Description: "Shows information about a player.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.Whois, ModeratorOnly: false},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "players-online",
		Description: "Shows detailed info about players currently online.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.PlayersOnline, ModeratorOnly: false},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "server-info",
		Description: "Shows detailed information on the server settings.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.ShowSettings, ModeratorOnly: false},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "register",
		Description: "Registers a new account, giving you accociated Discord roles with more privleges.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.AccessServer, ModeratorOnly: false},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "vote-rewind",
		Description: "Vote to rewind the map to the specified autosave (two votes needed!).",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.VoteRewind, ModeratorOnly: false},
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

func SlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	//Don't respond to other channels
	if i.AppID == cfg.Global.Discord.Application && i.ChannelID == cfg.Local.Channel.ChatChannel {
		for _, c := range CL {
			if c.AppCmd != nil && c.AppCmd.ID == data.ID {
				c.Command(s, i)
				return
			}
		}
	} else {
		fmt.Println("Ignoring command from non-chat channel:", i.Message.ChannelID)
	}
}
