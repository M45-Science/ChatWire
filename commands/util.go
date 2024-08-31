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
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
					if c.Function == nil {
						support.RunCommandOptions(&c, i)
					} else {
						c.Function(&c, i)
					}
					return
				}
			}
		}
	}
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
