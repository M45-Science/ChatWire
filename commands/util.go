package commands

import (
	"ChatWire/cfg"
	"ChatWire/commands/moderator"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"ChatWire/support"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var CommandLock sync.Mutex

func SlashCommand(unused *discordgo.Session, i *discordgo.InteractionCreate) {
	CommandLock.Lock()
	defer CommandLock.Unlock()

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
			//TODO clean these two options up
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

					fact.CheckVote(i, c)

					break
				}
			}
			for f, fType := range moderator.FTPTypes {
				if strings.EqualFold(data.CustomID, fType.Value) {
					if c == "INVALID" {
						disc.EphemeralResponse(i, "Error:", "Invalid file!")
						break
					}
					disc.EphemeralResponse(i, "Status:", "Loading "+fType.Name+": "+c)
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
						RunCommand(&c, i)
						var options []string
						for _, o := range c.AppCmd.Options {
							options = append(options, o.Name)
						}
						cwlog.DoLogCW("%s: ADMIN COMMAND: %s: %v", i.Member.User.Username, data.Name, strings.Join(options, ", "))
						return
					} else {
						disc.EphemeralResponse(i, "Error", "You must be a admin to use this command.")
						fact.CMS(i.ChannelID, "("+i.Member.User.Username+" does not have Discord admin permissions, and attempted to run the command: "+c.AppCmd.Name+")")
						return
					}
				} else if c.ModeratorOnly {
					if disc.CheckModerator(i) || disc.CheckAdmin(i) {
						cwlog.DoLogCW("%s: MOD COMMAND: %s", i.Member.User.Username, data.Name)
						RunCommand(&c, i)
						return
					} else {
						disc.EphemeralResponse(i, "Error", "You must be a moderator to use this command.")
						fact.CMS(i.ChannelID, "("+i.Member.User.Username+" does not have Discord moderator permissions, and attempted to run the command: "+c.AppCmd.Name+")")
						return
					}
				} else {
					cwlog.DoLogCW("%s: command: %s", i.Member.User.Username, data.Name)
					RunCommand(&c, i)
					return
				}
			}
		}
	}
}

func RunCommand(c *glob.CommandData, i *discordgo.InteractionCreate) {
	if c.Function == nil {
		support.RunCommandOptions(c, i)
	} else {
		c.Function(c, i)
	}
}

func DeregisterCommands() {
	if disc.DS == nil {
		return
	}
	if *glob.DoDeregisterCommands && disc.DS != nil {
		cmds, _ := disc.DS.ApplicationCommands(cfg.Global.Discord.Application, cfg.Global.Discord.Guild)
		for _, v := range cmds {
			cwlog.DoLogCW("Deregistered command: %s", v.Name)
			err := disc.DS.ApplicationCommandDelete(disc.DS.State.User.ID, cfg.Global.Discord.Guild, v.ID)
			if err != nil {
				cwlog.DoLogCW(err.Error())
			}

			time.Sleep(constants.ApplicationCommandSleep)
		}
		cwlog.DoLogCW("Deregister commands complete.")
	}
}

//https://discord.com/developers/docs/topics/permissions

var adminPerms int64 = discordgo.PermissionAdministrator     //Admin
var modPerms int64 = discordgo.PermissionManageRoles         //Manage Roles
var playerPerms int64 = discordgo.PermissionUseSlashCommands //Use slash comamnds

/*  RegisterCommands registers the commands on start up. */
func RegisterCommands() {

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

				tmpOption := &discordgo.ApplicationCommandOption{
					Name: option.Name, Description: option.Description, Type: option.Type, Required: option.Required, Choices: choiceList}

				if option.MinValue != nil {
					tmpOption.MinValue = option.MinValue
				}
				if option.MaxValue != nil {
					tmpOption.MaxValue = *option.MaxValue
				}

				tempAppCmd.Options = append(tempAppCmd.Options, tmpOption)
			}

			_, err := disc.DS.ApplicationCommandCreate(cfg.Global.Discord.Application, cfg.Global.Discord.Guild, tempAppCmd)
			if err != nil {
				log.Println("Failed to create command: ",
					CL[i].AppCmd.Name, ": ", err)
				continue
			}
			cwlog.DoLogCW("Registered command: %s", CL[i].AppCmd.Name)
		}
		cwlog.DoLogCW("Register commands complete.")
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
				MinValue:    glob.Ptr(float64(o.MinInt)),
				MaxValue:    glob.Ptr(float64(o.MaxInt)),
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
				MinValue:    glob.Ptr(float64(o.MinF32)),
				MaxValue:    glob.Ptr(float64(o.MaxF32)),
			})
		} else if o.Type == moderator.TYPE_F64 {
			CL[p].AppCmd.Options = append(CL[p].AppCmd.Options, glob.OptionData{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        filterName(o.Name),
				Description: filterDesc(o.Desc),
				MinValue:    glob.Ptr(o.MinF64),
				MaxValue:    glob.Ptr(o.MaxF64),
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
