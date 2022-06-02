package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/commands/admin"
	"ChatWire/commands/moderator"
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
	AppCmd        *discordgo.ApplicationCommand
	ModeratorOnly bool
	AdminOnly     bool

	PrimaryOnly bool
}

var CL []Command
var BugOne float64 = 1

var cmds = []Command{

	/* Admin Commands */
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "chatwire",
		Description: "Actions specific to ChatWire.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "action",
				Description: "do not use these unless you are certain of what you are doing",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "reboot",
						Value: "reboot",
					},
					{
						Name:  "queue-reboot",
						Value: "queue-reboot",
					},
					{
						Name:  "force-reboot",
						Value: "force-reboot",
					},
					{
						Name:  "reload-config",
						Value: "reload-config",
					},
				},
			},
		},
	},
		Command: admin.ChatWire, AdminOnly: true},
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "map-schedule",
		Description: "Set a map reset schedule.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "preset",
				Description: "How often to reset the map, based on a preset.",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
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
						Name:  "fridays",
						Value: "fridays",
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
		},
	},
		Command: admin.SetSchedule, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "factorio",
		Description: "Actions specific to Factorio.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "action",
				Description: "do not use these unless you are certain of what you are doing",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "start",
						Value: "start",
					},
					{
						Name:  "stop",
						Value: "stop",
					},
					{
						Name:  "new-map-preview",
						Value: "new-map-preview",
					},
					{
						Name:  "new-map",
						Value: "new-map",
					},
					{
						Name:  "update-factorio",
						Value: "update-factorio",
					},
					{
						Name:  "update-mods",
						Value: "update-mods",
					},
					{
						Name:  "archive-map",
						Value: "archive-map",
					},
					{
						Name:  "install-factorio",
						Value: "install-factorio",
					},
				},
			},
		},
	},
		Command: admin.Factorio, AdminOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "config-global",
		Description: "Global server settings and options.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: admin.GConfigServer, AdminOnly: true, PrimaryOnly: true},
	/* MODERATOR COMMANDS ---------------- */
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "rcon",
		Description: "remote console (remotely run a factorio command)",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "command",
				Description: "factorio command to run",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
		Command: moderator.RCONCmd, ModeratorOnly: true},
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "map-reset",
		Description: "automated map reset, will kick players out of game.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: moderator.MapReset, ModeratorOnly: true},
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "config-server",
		Description: "Server settings and options.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: moderator.ConfigServer, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "player-level",
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
		Command: moderator.PlayerLevel, ModeratorOnly: true},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "change-map",
		Description: "Load a specific saved map.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: moderator.ChangeMap, ModeratorOnly: true},

	/* PLAYER COMMMANDS -------------------- */
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "info",
		Description: "displays status and settings of the server.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "options",
				Description: "verbose shows all info, instead of just relevant info. debug is for dev use only.",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "verbose",
						Value: "verbose",
					},
					{
						Name:  "debug",
						Value: "debug",
					},
				},
			},
		},
	},
		Command: user.Info},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "players",
		Description: "Shows detailed info about players currently online.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.Players},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "vote-map",
		Description: "Vote for a new map, or a previous map. Requires TWO votes, requires `" + strings.ToUpper(cfg.Global.Discord.Roles.Regular) + "` discord role.",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "moderator",
				Description: "moderator only options",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
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
		Command: user.VoteMap},

	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "register",
		Description: "Registers a new account, giving you associated Discord roles with more privleges.",
		Type:        discordgo.ChatApplicationCommand,
	},
		Command: user.Register, PrimaryOnly: true},
	{AppCmd: &discordgo.ApplicationCommand{
		Name:        "whois",
		Description: "Shows information about <search>",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "search",
				Description: "Factorio/Discord name, or any partial match.",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
		Command: user.Whois, PrimaryOnly: true},
}

func ClearCommands() {
	if *glob.DoDeregisterCommands && disc.DS != nil {
		cmds, _ := disc.DS.ApplicationCommands(cfg.Global.Discord.Application, cfg.Global.Discord.Guild)
		for _, v := range cmds {
			cwlog.DoLogCW(fmt.Sprintf("Deregistered command: %s", v.Name))
			err := disc.DS.ApplicationCommandDelete(disc.DS.State.User.ID, cfg.Global.Discord.Guild, v.ID)
			if err != nil {
				cwlog.DoLogCW(err.Error())
			}

			time.Sleep(constants.ApplicationCommandSleep)
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

			if strings.EqualFold(o.AppCmd.Name, "config-server") {
				LinkConfigData(i, false)
			}
			if strings.EqualFold(o.AppCmd.Name, "config-global") {
				LinkConfigData(i, true)
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

func LinkConfigData(p int, gconfig bool) {

	var selection []moderator.SettingListData
	if gconfig {
		selection = moderator.GSettingList
	} else {
		selection = moderator.SettingList
	}
	for i, o := range selection {
		if i > 25 {
			cwlog.DoLogCW("LinkConfigData: Max 25 settings reached!")
			break
		}
		if o.Type == moderator.TYPE_STRING {

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

		} else if o.Type == moderator.TYPE_INT {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    glob.Ptr(float64(o.MinInt)),
				MaxValue:    float64(o.MaxInt),
			})
		} else if o.Type == moderator.TYPE_BOOL {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
			})
		} else if o.Type == moderator.TYPE_F32 {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    glob.Ptr(float64(o.MinF32)),
				MaxValue:    float64(o.MaxF32),
			})
		} else if o.Type == moderator.TYPE_F64 {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    glob.Ptr(o.MinF64),
				MaxValue:    o.MaxF64,
			})
		} else if o.Type == moderator.TYPE_CHANNEL {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
			})
		}
	}
}

func SlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {

	/* Ignore appid that aren't relevant to us */
	if i.AppID != cfg.Global.Discord.Application {
		return
	}

	if i.Member == nil {
		cwlog.DoLogCW("SlashCommand: Ignoring interaction with no member (dm).")
		return
	}

	if i.Type == discordgo.InteractionMessageComponent {
		data := i.MessageComponentData()

		for _, c := range data.Values {
			if strings.EqualFold(data.CustomID, "ChangeMap") {
				if disc.CheckModerator(i) || disc.CheckAdmin(i) {

					buf := fmt.Sprintf("Loading: %v, please wait.", c)
					elist := discordgo.MessageEmbed{Title: "Notice:", Description: buf}
					disc.InteractionResponse(s, i, &elist)

					fact.DoChangeMap(s, c)

					break
				}
			} else if strings.EqualFold(data.CustomID, "VoteMap") {
				if disc.CheckRegular(i) || disc.CheckModerator(i) || disc.CheckAdmin(i) {

					buf := fmt.Sprintf("Submitting vote for %v, one moment please.", c)
					disc.EphemeralResponse(s, i, "Notice:", buf)

					go fact.CheckVote(s, i, c)

					break
				}
			}
		}
	} else if i.Type == discordgo.InteractionApplicationCommand {
		data := i.ApplicationCommandData()

		for _, c := range CL {

			/* Hanadle PrimaryOnly commands if we are the primary, otherwise only allow commands from our channel */
			if !c.PrimaryOnly && !strings.EqualFold(i.ChannelID, cfg.Local.Channel.ChatChannel) {
				continue
			} else if c.PrimaryOnly && !strings.EqualFold(cfg.Local.Callsign, cfg.Global.PrimaryServer) {
				continue
			}

			if strings.EqualFold(c.AppCmd.Name, data.Name) {

				if c.AdminOnly {
					if disc.CheckAdmin(i) {
						c.Command(s, i)
						var options []string
						for _, o := range c.AppCmd.Options {
							options = append(options, o.Name)
						}
						cwlog.DoLogCW(fmt.Sprintf("%s: ADMIN COMMAND: %s: %v", i.Member.User.Username, data.Name, strings.Join(options, ", ")))
						return
					} else {
						disc.EphemeralResponse(s, i, "Error", "You must be a admin to use this command.")
						fact.CMS(i.ChannelID, "You do not have permission to use admin commands. ("+i.Member.User.Username+", "+c.AppCmd.Name+")")
						return
					}
				} else if c.ModeratorOnly {
					if disc.CheckModerator(i) || disc.CheckAdmin(i) {
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
	}
}
