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
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

type Command struct {
	Command       func(s *discordgo.Session, i *discordgo.InteractionCreate)
	ModeratorOnly bool
	AdminOnly     bool
	AppCmd        *discordgo.ApplicationCommand
	PrimaryOnly   bool
}

var CL []Command
var BugOne float64 = 1

var cmds = []Command{

	/* Admin Commands */

	//Make "reboot" command with all of these contained __START__
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "stop-factorio",
		Description: "Stops Factorio, if running.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.StopFact, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "start-factorio",
		Description: "Starts OR restarts Factorio, even if already running.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.StartFact, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "reboot-chatwire",
		Description: "Closes Factorio (if running), and restarts ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.RebootCW, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "force-reboot-chatwire",
		Description: "Big red button. Don't use this lightly. This does not cleanly exit Factorio or ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ForceReboot, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "queue-reboot",
		Description: "Queues up a reboot. This waits until no players are online to reboot Factorio and ChatWire.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.QueReboot, AdminOnly: true},
	//Make "reboot" command with all of these contained __END__

	//Put all these in a map command
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "archive-map",
		Description: "Archives the current map to our website, and posts the link to the chat.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ArchiveMap, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "new-map-preview",
		Description: "Posts a new map, with preview to discord. Use /make-new-map after to create it.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.NewMapPrev, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "make-new-map",
		Description: "Creates a new map.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.MakeNewMap, AdminOnly: true},

	//Put this in a "update" command
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
		Command: admin.UpdateFact, AdminOnly: true},
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "reload-config",
		Description: "Reloads config files from disk, only used when manually editing config files.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.ReloadConfig, AdminOnly: true},
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "debug",
		Description: "Only used for development and testing.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.DebugStat, AdminOnly: true},
	//Put this in a "update" command
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "update-mods",
		Description: "Updates Factorio mods to the latest version if there is a new version available.",
	},
		Command: admin.UpdateMods, AdminOnly: true},

	/*  Moderator Commands ------------- */
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "config",
		Description: "Change server configuration options.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.Config, ModeratorOnly: true},
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "map-reset",
		Description: "Stops Factorio, archives current map, generates new one, and starts Factorio.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.MapReset, ModeratorOnly: true},

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
		Command: admin.SetPlayerLevel, ModeratorOnly: true, PrimaryOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "rewind-map",
		Description: "Rewinds the map to specified autosave.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "rewind-to",
				Description: "autosave to rewind to",
				Required:    false,
				MinValue:    glob.Ptr(float64(0)),
				MaxValue:    float64(cfg.Global.Options.AutosaveMax),
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "list",
				Description: "Show list of the last 40 autosaves",
				Required:    false,
			},
		},
	},
		Command: admin.RewindMap, ModeratorOnly: true},

	/* Move to config */
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "set-map-seed",
		Description: "Sets the map seed for the next map reset. Value is cleared after use.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.SetSeed, ModeratorOnly: true},

	/* PLAYER COMMMANDS -------------------- */
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "whois",
		Description: "Shows information about a player.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.Whois, ModeratorOnly: false, PrimaryOnly: true},

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
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "verbose",
				Description: "Show everything, instead of just relevant info.",
				Required:    false,
			},
		},
	},
		Command: user.ServerInfo, ModeratorOnly: false},

	//Cleanup, possibly handle other chat channels
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "register",
		Description: "Registers a new account, giving you associated Discord roles with more privleges.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.Register, ModeratorOnly: false, PrimaryOnly: true},

	//Add params, make slicker
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "vote-rewind",
		Description: "Vote to rewind the map to the specified autosave (two votes needed!).",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "autosave",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Description: "The number of the autosave to rewind to.",
				MinValue:    &BugOne,
				MaxValue:    float64(cfg.Global.Options.AutosaveMax),
			},
		},
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

				time.Sleep(constants.ApplicationCommandSleep)
			}
		}
	}
}

//https://discord.com/developers/docs/topics/permissions

var adminPerms int64 = (1 << 3)   //Administrator
var modPerms int64 = (1 << 28)    //MANAGE_ROLES
var playerPerms int64 = (1 << 11) //SEND_MESSAGES

/*  RegisterCommands registers the commands on start up. */
func RegisterCommands(s *discordgo.Session) {

	/* Bypasses init loop compile error. */
	CL = cmds

	//Bypass register, very slow
	//TODO: Cache info and correct for changes when needed

	if *glob.DoRegisterCommands {

		for i, o := range CL {

			if o.AppCmd == nil {
				continue
			}
			if o.AppCmd.Name == "" || o.AppCmd.Description == "" {
				cwlog.DoLogCW("Command has no name or description, skipping")
				continue
			}
			time.Sleep(constants.ApplicationCommandSleep)

			if strings.EqualFold(o.AppCmd.Name, "config") {
				LinkConfigData(i)
			}

			if o.AdminOnly {
				o.AppCmd.DefaultMemberPermissions = &adminPerms
			} else if o.ModeratorOnly {
				o.AppCmd.DefaultMemberPermissions = &modPerms
			} else {
				o.AppCmd.DefaultMemberPermissions = &playerPerms
			}

			o.AppCmd.Name = filterName(o.AppCmd.Name)
			o.AppCmd.Description = filterDesc(o.AppCmd.Description)

			cmd, err := s.ApplicationCommandCreate(cfg.Global.Discord.Application, cfg.Global.Discord.Guild, o.AppCmd)
			if err != nil {
				log.Println("Failed to create command: ",
					CL[i].AppCmd.Name, ": ", err)
				continue
			}
			CL[i].AppCmd = cmd
			cwlog.DoLogCW(fmt.Sprintf("Registered command: %s", CL[i].AppCmd.Name))
		}
	}

}

func filterName(name string) string {
	newName := strings.ToLower(name)
	newName = strings.Replace(newName, " ", "-", -1)
	newName = sclean.TruncateString(newName, 32)

	return newName
}

func filterDesc(desc string) string {
	newDesc := sclean.TruncateStringEllipsis(desc, 100)

	if len(desc) > 0 {
		return newDesc
	} else {
		buf := "No description available."
		return buf
	}
}

func LinkConfigData(p int) {

	for i, o := range admin.SettingList {
		if i > 25 {
			cwlog.DoLogCW("LinkConfigData: Max 25 settings reached!")
			break
		}
		if o.Type == admin.TYPE_STRING {

			if len(o.ValidStrings) > 0 {
				choices := []*discordgo.ApplicationCommandOptionChoice{}
				for _, v := range o.ValidStrings {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  filterName(v),
						Value: filterName(v),
					})
				}

				if len(choices) > 0 {
					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
						Choices:     choices,
					})
				} else {
					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
					})
				}
			} else if o.ListString != nil {
				choices := []*discordgo.ApplicationCommandOptionChoice{}
				list := o.ListString()
				for _, v := range list {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  filterName(v),
						Value: filterName(v),
					})
				}

				if len(choices) > 0 {

					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
						Choices:     choices,
					})
				} else {
					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
					})
				}
			} else {
				CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        filterName(o.Name),
					Description: filterDesc(o.Desc),
				})
			}

		} else if o.Type == admin.TYPE_INT {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    glob.Ptr(float64(o.MinInt)),
				MaxValue:    float64(o.MaxInt),
			})
		} else if o.Type == admin.TYPE_BOOL {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
			})
		} else if o.Type == admin.TYPE_F32 {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    glob.Ptr(float64(o.MinF32)),
				MaxValue:    float64(o.MaxF32),
			})
		} else if o.Type == admin.TYPE_F64 {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    glob.Ptr(o.MinF64),
				MaxValue:    o.MaxF64,
			})
		}
	}
}

func SlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	/* Ignore events and appid that aren't relevant to us */
	if i.Type != discordgo.InteractionApplicationCommand || i.AppID != cfg.Global.Discord.Application {
		return
	}

	data := i.ApplicationCommandData()

	/* Handle these commands from anywhere, if we are the primary server */
	if strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) {
		for _, c := range CL {
			if strings.EqualFold(c.AppCmd.Name, data.Name) {
				if c.PrimaryOnly {
					c.Command(s, i)
					return
				}
			}
		}
	}

	/* Don't respond to other channels for normal commands */
	if i.ChannelID == cfg.Local.Channel.ChatChannel {

		for _, c := range CL {

			/* Don't process these commands here */
			if c.PrimaryOnly {
				continue
			}

			if strings.EqualFold(c.AppCmd.Name, data.Name) {

				if c.AdminOnly {
					if disc.CheckAdmin(i.Member.Roles) {
						c.Command(s, i)
						cwlog.DoLogCW(fmt.Sprintf("%s: ADMIN COMMAND: %s", i.Member.User.Username, data.Name))
						return
					} else {
						disc.EphemeralResponse(s, i, "Error", "You must be a admin to use this command.")
						fact.CMS(i.ChannelID, "You do not have permission to use admin commands. ("+i.Member.User.Username+", "+c.AppCmd.Name+")")
						return
					}
				} else if c.ModeratorOnly {
					if disc.CheckModerator(i.Member.Roles) {
						cwlog.DoLogCW(fmt.Sprintf("%s: MOD COMMAND: %s", i.Member.User.Username, data.Name))
						c.Command(s, i)
						return
					} else {
						disc.EphemeralResponse(s, i, "Error", "You must be a moderator to use this command.")
						fact.CMS(i.ChannelID, "You do not have permission to use moderator commands. ("+i.Member.User.Username+", "+c.AppCmd.Name+")")
						return
					}
				} else {
					cwlog.DoLogCW(fmt.Sprintf("%s: command: %s", i.Member.User.Username, data.Name))
					c.Command(s, i)
					return
				}
			}
		}
		disc.EphemeralResponse(s, i, "Error", "That is not a valid command.")
		cwlog.DoLogCW(fmt.Sprintf("INVALID COMMAND: %s: command: %s", i.Member.User.Username, data.Name))

	}
}
