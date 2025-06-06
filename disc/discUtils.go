package disc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/glob"
)

/*  Check if Discord admin */
func CheckAdmin(i *discordgo.InteractionCreate) bool {

	if cfg.Global.Discord.Roles.RoleCache.Admin == "" {
		cwlog.DoLogCW("CheckAdmin: RoleID not found for that role, check configuration files.")
		return false
	}

	if i.Member != nil {
		for _, r := range i.Member.Roles {
			if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Admin) {
				return true
			}
		}
	}
	return false
}

/*  Check if Discord moderator */
func CheckModerator(i *discordgo.InteractionCreate) bool {

	if cfg.Global.Discord.Roles.RoleCache.Moderator == "" {
		cwlog.DoLogCW("CheckModerator: RoleID not found for that role, check configuration files.")
		return false
	}

	if i.Member != nil {
		for _, r := range i.Member.Roles {
			if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Moderator) {
				return true
			}
		}
	}
	return false
}

func CheckSupporter(i *discordgo.InteractionCreate) bool {

	if cfg.Global.Discord.Roles.RoleCache.Patreon == "" ||
		cfg.Global.Discord.Roles.RoleCache.Supporter == "" {
		cwlog.DoLogCW("CheckSupporter RoleID not found for that role, check configuration files.")
		return false
	}

	if i.Member != nil {
		for _, r := range i.Member.Roles {
			if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Patreon) ||
				strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Supporter) {
				return true
			}
		}
	}
	return false
}

/* Check if Discord regular */
func CheckRegular(i *discordgo.InteractionCreate) bool {

	if cfg.Global.Discord.Roles.RoleCache.Regular == "" {
		cwlog.DoLogCW("CheckRegular RoleID not found for that role, check configuration files.")
		return false
	}

	if i.Member != nil {
		for _, r := range i.Member.Roles {
			if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Regular) {
				return true
			}
		}
	}
	return false
}

func CheckVeteran(i *discordgo.InteractionCreate) bool {

	if cfg.Global.Discord.Roles.RoleCache.Veteran == "" {
		cwlog.DoLogCW("CheckVeteran RoleID not found for that role, check configuration files.")
		return false
	}

	if i.Member != nil {
		for _, r := range i.Member.Roles {
			if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Veteran) {
				return true
			}
		}
	}
	return false
}

/* Check if Discord member */
func CheckMember(i *discordgo.InteractionCreate) bool {

	if cfg.Global.Discord.Roles.RoleCache.Member == "" {
		cwlog.DoLogCW("CheckMember RoleID not found for that role, check configuration files.")
		return false
	}

	if i.Member != nil {
		for _, r := range i.Member.Roles {
			if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Member) {
				return true
			}
		}
	}
	return false
}

/* Check if Discord member */
func CheckNew(i *discordgo.InteractionCreate) bool {

	if cfg.Global.Discord.Roles.RoleCache.New == "" {
		cwlog.DoLogCW("CheckNew: RoleID not found for that role, check configuration files.")
		return false
	}

	if i.Member != nil {
		for _, r := range i.Member.Roles {
			if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.New) {
				return true
			}
		}
	}
	return false
}

/* Send embedded message */
func SmartWriteDiscordEmbed(ch string, embed *discordgo.MessageEmbed) *discordgo.Message {

	if ch == "" || embed == nil {
		return nil
	}

	if DS != nil {
		msg, err := DS.ChannelMessageSendEmbed(ch, embed)

		if err != nil {
			cwlog.DoLogCW("SmartWriteDiscordEmbed: ERROR: %v", err)
		}
		return msg
	}

	return nil
}

/* Send embedded message */
func SmartEditDiscordEmbed(ch string, msg *discordgo.Message, title, description string, color int) *discordgo.Message {

	if ch == "" {
		return nil
	}

	if DS != nil {
		if msg != nil && msg.ID != "" && len(msg.Embeds) > 0 {
			embed := msg.Embeds[0]
			embed.Title = title
			embed.Description = embed.Description + "\n" + description
			embed.Color = color

			msg, err := DS.ChannelMessageEditEmbed(msg.ChannelID, msg.ID, embed)
			if err != nil {
				msg = SmartWriteDiscordEmbed(ch, &discordgo.MessageEmbed{Title: title, Description: description, Color: color})
			}
			return msg
		} else {
			return SmartWriteDiscordEmbed(ch, &discordgo.MessageEmbed{Title: title, Description: description, Color: color})
		}
	}

	return nil
}

/*Send normal message to a channel*/
func SmartWriteDiscord(ch string, text string) *discordgo.Message {

	if ch == "" || text == "" {
		return nil
	}

	if DS != nil {
		msg, err := DS.ChannelMessageSend(ch, text)

		if err != nil {
			cwlog.DoLogCW("SmartWriteDiscord: ERROR: %v", err)
		}
		return msg
	}
	return nil
}

/* Give a player a role */
func SmartRoleAdd(gid string, uid string, rid string) error {

	if DS != nil {
		err := DS.GuildMemberRoleAdd(gid, uid, rid)

		if err != nil {
			if !strings.Contains(err.Error(), "Unknown Member") {
				cwlog.DoLogCW("SmartRoleAdd: ERROR: %v", err)
			} else {
				return nil
			}
		}
		return err
	}

	return errors.New("error")
}

/* See if a role exists */
func RoleExists(g *discordgo.Guild, name string) (bool, *discordgo.Role) {

	if g != nil && name != "" {
		name = strings.ToLower(name)

		for _, role := range g.Roles {
			if strings.EqualFold(role.Name, "@everyone") {
				continue
			}

			if strings.EqualFold(role.Name, name) {
				return true, role
			}

		}

	}
	return false, nil
}

/* Discord name from discordid */
func GetNameFromID(id string) string {
	if id == "" || DS == nil {
		return ""
	}
	g := Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				return m.User.Username
			}
		}
	}

	return ""
}

/* Discord avatar from discordid */
func GetDiscordAvatarFromId(id string, size int) string {

	if id == "" || DS == nil {
		return ""
	}
	g := Guild

	if g != nil {
		for _, m := range g.Members {
			if strings.EqualFold(m.User.ID, id) {
				return m.User.AvatarURL(fmt.Sprintf("%v", size))
			}
		}
	}

	return ""
}

/* Look up DiscordID, from Factorio name. Only works for players that have registered */
func GetDiscordIDFromFactorioName(input string) string {

	pname := strings.ToLower(input)

	if pname == "" {
		return ""
	}

	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()

	if glob.PlayerList[pname] != nil && glob.PlayerList[pname].Level > 0 {
		return glob.PlayerList[pname].ID
	}
	return ""
}

/* Look up Factorio name, from DiscordID. Only works for players that have registered */
func GetFactorioNameFromDiscordID(id string) string {

	if id == "" {
		return ""
	}

	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()

	for i, player := range glob.PlayerList {
		if strings.EqualFold(glob.PlayerList[i].ID, id) && player.Level > 0 {
			return player.Name
		}
	}
	return ""
}

func GetPlayerDataFromName(input string) *glob.PlayerData {
	pname := strings.ToLower(input)

	if pname == "" {
		return nil
	}

	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()

	p := glob.PlayerList[pname]
	return p
}

func InteractionEphemeralResponse(i *discordgo.InteractionCreate, title, message string) *discordgo.Message {
	return InteractionEphemeralResponseColor(i, title, message, glob.COLOR_WHITE)
}

func InteractionEphemeralResponseColor(i *discordgo.InteractionCreate, title, message string, color int) *discordgo.Message {
	glob.BootMessage = nil
	glob.UpdateMessage = nil

	if DS == nil {
		return nil
	}
	cwlog.DoLogCW("EphemeralResponse:\n" + i.Member.User.Username + "\n" + title + "\n" + message)

	embed := []*discordgo.MessageEmbed{{Title: title, Description: message, Color: color}}
	if i.Interaction != nil {
		resp := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: embed, Flags: discordgo.MessageFlagsEphemeral},
		}
		err := DS.InteractionRespond(i.Interaction, resp)
		if err != nil {
			newResp := &discordgo.WebhookEdit{Embeds: &embed}
			DS.InteractionResponseEdit(i.Interaction, newResp)
		}
		return nil
	}
	msg, err := DS.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{Embeds: embed, Flags: discordgo.MessageFlagsEphemeral})
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
	return msg
}
