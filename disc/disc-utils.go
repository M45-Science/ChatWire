package disc

import (
	"fmt"
	"strings"
	"time"

	"ChatWire/botlog"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

/* Send embeded message */
func SmartWriteDiscordEmbed(ch string, embed *discordgo.MessageEmbed) error {

	if ch == "" || embed == nil {
		return nil
	}

	if DS != nil {
		_, err := DS.ChannelMessageSendEmbed(ch, embed)

		if err != nil {

			botlog.DoLog(fmt.Sprintf("SmartWriteDiscordEmbed: ERROR: %v", err))
		}

		return err
	}

	return fmt.Errorf("error")
}

/*Send normal message to a channel*/
func SmartWriteDiscord(ch string, text string) {

	if ch == "" || text == "" {
		return
	}

	if DS != nil {
		_, err := DS.ChannelMessageSend(ch, text)

		if err != nil {

			botlog.DoLog(fmt.Sprintf("SmartWriteDiscord: ERROR: %v", err))
		}
	} else {

		time.Sleep(5 * time.Second)
		SmartWriteDiscord(ch, text)
	}
}

/* Create a Discord channel */
func SmartChannelCreate(id string) *discordgo.Channel {

	if DS != nil {
		ch, err := DS.UserChannelCreate(id)

		if err != nil || ch == nil {

			botlog.DoLog(fmt.Sprintf("SmartChannelCreate: ERROR: %v", err))
		} else {
			return ch
		}
	} else {

		time.Sleep(30 * time.Second)
		SmartChannelCreate(id)
	}

	return nil
}

/* Give a user a role */
func SmartRoleAdd(gid string, uid string, rid string) error {

	if DS != nil {
		err := DS.GuildMemberRoleAdd(gid, uid, rid)

		if err != nil {

			botlog.DoLog(fmt.Sprintf("SmartRoleAdd: ERROR: %v", err))
		}

		return err
	}

	return fmt.Errorf("error")
}

/* See if a role exists */
func RoleExists(g *discordgo.Guild, name string) (bool, *discordgo.Role) {

	if g != nil && name != "" {
		name = strings.ToLower(name)

		for _, role := range g.Roles {
			if role.Name == "@everyone" {
				continue
			}

			if strings.ToLower(role.Name) == name {
				return true, role
			}

		}

	}
	return false, nil
}

//Discord name from discordid
func GetNameFromID(id string, disc bool) string {
	if id == "" || DS == nil {
		return ""
	}
	g := Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				if disc {
					return m.User.Username + "#" + m.User.Discriminator
				} else {
					return m.User.Username
				}
			}
		}
	}

	return id
}

//Discord avatar from discordid
func GetDiscordAvatarFromId(id string, size int) string {

	if id == "" || DS == nil {
		return ""
	}
	g := Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				return m.User.AvatarURL(fmt.Sprintf("%v", size))
			}
		}
	}

	return ""
}

/* Look up DiscordID, from Factorio name. Only works for players that have registered */
func GetDiscordIDFromFactorioName(pname string) string {

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
		if glob.PlayerList[i].ID == id && player.Level > 0 {
			return player.Name
		}
	}
	return ""
}
