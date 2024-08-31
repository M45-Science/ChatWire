package commands

import (
	"fmt"
	"log"
	"os"
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
		Function: admin.ChatWire, AdminOnly: true},
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
		Function: admin.Factorio, AdminOnly: true},

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
		Function: moderator.FTPLoad, ModeratorOnly: true},
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

func DeregisterCommands() {
	if disc.DS == nil {
		return
	}
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
	_ = os.Remove("cw.lock")
	os.Exit(0)
}

//https://discord.com/developers/docs/topics/permissions

var adminPerms int64 = discordgo.PermissionAdministrator     //Admin
var modPerms int64 = discordgo.PermissionManageRoles         //Manage Roles
var playerPerms int64 = discordgo.PermissionUseSlashCommands //Use slash comamnds

/*  RegisterCommands registers the commands on start up. */
func RegisterCommands(s *discordgo.Session) {

	/* Bypasses init loop compile error. */
	CL = cmds

	//Bypass register, very slow
	//TODO: Cache info and correct for changes when needed

	if *glob.DoRegisterCommands {

		for i, o := range CL {

			if o.Disabled {
				continue
			}
			if o.AppCmd.Name == "" || o.AppCmd.Description == "" {
				cwlog.DoLogCW("Command has no name or description, skipping")
				continue
			}
			time.Sleep(constants.ApplicationCommandSleep)

			// TODO not working
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

			//Convert local format to discord format
			tempAppCmd := &discordgo.ApplicationCommand{
				Name: o.AppCmd.Name, Description: o.AppCmd.Description, Type: o.AppCmd.Type, DefaultMemberPermissions: o.AppCmd.DefaultMemberPermissions,
				Options: []*discordgo.ApplicationCommandOption{},
			}

			for _, option := range CL[i].AppCmd.Options {
				var choiceList []*discordgo.ApplicationCommandOptionChoice
				for _, choice := range option.Choices {
					choiceList = append(choiceList, &discordgo.ApplicationCommandOptionChoice{Name: choice.Name, Value: choice.Value})
				}
				tempAppCmd.Options = append(tempAppCmd.Options, &discordgo.ApplicationCommandOption{
					Name: option.Name, Description: option.Description, Type: option.Type, Required: option.Required, MinValue: glob.Ptr(option.MinValue), MaxValue: option.MaxValue, Choices: choiceList})
			}

			cmd, err := s.ApplicationCommandCreate(cfg.Global.Discord.Application, cfg.Global.Discord.Guild, tempAppCmd)
			if err != nil {
				log.Println("Failed to create command: ",
					CL[i].AppCmd.Name, ": ", err)
				continue
			}
			CL[i].DiscCmd = cmd
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

// TODO not working
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
				choices := []glob.ChoiceData{}
				for _, v := range o.ValidStrings {
					choices = append(choices, glob.ChoiceData{
						Name:  filterName(v),
						Value: filterName(v),
					})
				}

				if len(choices) > 0 {
					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
						Choices:     choices,
					})
				} else {
					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
					})
				}
			} else if o.ListString != nil {
				choices := []glob.ChoiceData{}
				list := o.ListString()
				for _, v := range list {
					choices = append(choices, glob.ChoiceData{
						Name:  filterName(v),
						Value: filterName(v),
					})
				}

				if len(choices) > 0 {

					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
						Choices:     choices,
					})
				} else {
					CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        filterName(o.Name),
						Description: filterDesc(o.Desc),
					})
				}
			} else {
				CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        filterName(o.Name),
					Description: filterDesc(o.Desc),
				})
			}

		} else if o.Type == moderator.TYPE_INT {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    float64(o.MinInt),
				MaxValue:    float64(o.MaxInt),
			})
		} else if o.Type == moderator.TYPE_BOOL {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
			})
		} else if o.Type == moderator.TYPE_F32 {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    float64(o.MinF32),
				MaxValue:    float64(o.MaxF32),
			})
		} else if o.Type == moderator.TYPE_F64 {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    o.MinF64,
				MaxValue:    o.MaxF64,
			})
		} else if o.Type == moderator.TYPE_CHANNEL {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
			})
		}
	}
}

func SlashCommand(unused *discordgo.Session, i *discordgo.InteractionCreate) {

	/* Ignore appid that aren't relevant to us */
	if i.AppID != cfg.Global.Discord.Application {
		return
	}

	if i.GuildID != cfg.Global.Discord.Guild {
		return
	}

	if i.Member == nil {
		cwlog.DoLogCW("SlashCommand: Ignoring interaction with no member (dm).")
		return
	}

	if i.Type == discordgo.InteractionMessageComponent &&
		strings.EqualFold(i.ChannelID, cfg.Local.Channel.ChatChannel) {
		data := i.MessageComponentData()

		for _, c := range data.Values {
			if strings.EqualFold(data.CustomID, "ChangeMap") {
				if disc.CheckModerator(i) || disc.CheckAdmin(i) {

					buf := fmt.Sprintf("Loading: %v, please wait.", c)
					elist := discordgo.MessageEmbed{Title: "Notice:", Description: buf}
					disc.InteractionResponse(i, &elist)

					fact.DoChangeMap(c)

					break
				}
			} else if strings.EqualFold(data.CustomID, "VoteMap") {
				if disc.CheckRegular(i) || disc.CheckModerator(i) || disc.CheckAdmin(i) {

					buf := fmt.Sprintf("Submitting vote for %v, one moment please.", c)
					disc.EphemeralResponse(i, "Notice:", buf)

					go fact.CheckVote(i, c)

					break
				}
			}
			for f, fType := range moderator.FTPTypes {
				if strings.EqualFold(data.CustomID, fType.ID) {
					if c == "INVALID" {
						disc.EphemeralResponse(i, "Error:", "Invalid file!")
						break
					}
					//disc.EphemeralResponse( i, "Status:", "Loading "+fType.Name+": "+c)
					moderator.LoadFTPFile(i, c, f)
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
						c.Function(&c, i)
						var options []string
						for _, o := range c.AppCmd.Options {
							options = append(options, o.Name)
						}
						cwlog.DoLogCW(fmt.Sprintf("%s: ADMIN COMMAND: %s: %v", i.Member.User.Username, data.Name, strings.Join(options, ", ")))
						return
					} else {
						disc.EphemeralResponse(i, "Error", "You must be a admin to use this command.")
						fact.CMS(i.ChannelID, "("+i.Member.User.Username+" does not have Discord admin permissions, and attempted to run the command: "+c.AppCmd.Name+")")
						return
					}
				} else if c.ModeratorOnly {
					if disc.CheckModerator(i) || disc.CheckAdmin(i) {
						cwlog.DoLogCW(fmt.Sprintf("%s: MOD COMMAND: %s", i.Member.User.Username, data.Name))
						c.Function(&c, i)
						return
					} else {
						disc.EphemeralResponse(i, "Error", "You must be a moderator to use this command.")
						fact.CMS(i.ChannelID, "("+i.Member.User.Username+" does not have Discord moderator permissions, and attempted to run the command: "+c.AppCmd.Name+")")
						return
					}
				} else {
					cwlog.DoLogCW(fmt.Sprintf("%s: command: %s", i.Member.User.Username, data.Name))
					c.Function(&c, i)
					return
				}
			}
		}
	}
}
