package disc

import (
	"fmt"
	"strings"
	"time"

	"ChatWire/botlog"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

func SmartWriteDiscordEmbed(ch string, embed *discordgo.MessageEmbed) error {

	if ch == "" || embed == nil {
		return nil
	}

	if glob.DS != nil {
		_, err := glob.DS.ChannelMessageSendEmbed(ch, embed)

		if err != nil {

			botlog.DoLog(fmt.Sprintf("SmartWriteDiscordEmbed: ERROR: %v", err))
		}

		return err
	}

	return fmt.Errorf("error")
}

func SmartWriteDiscord(ch string, text string) {

	if ch == "" || text == "" {
		return
	}

	if glob.DS != nil {
		_, err := glob.DS.ChannelMessageSend(ch, text)

		if err != nil {

			botlog.DoLog(fmt.Sprintf("SmartWriteDiscord: ERROR: %v", err))
		}
	} else {

		time.Sleep(5 * time.Second)
		SmartWriteDiscord(ch, text)
	}
}

func SmartChannelCreate(id string) *discordgo.Channel {

	if glob.DS != nil {
		ch, err := glob.DS.UserChannelCreate(id)

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

func SmartRoleAdd(gid string, uid string, rid string) error {

	if glob.DS != nil {
		err := glob.DS.GuildMemberRoleAdd(gid, uid, rid)

		if err != nil {

			botlog.DoLog(fmt.Sprintf("SmartRoleAdd: ERROR: %v", err))
		}

		return err
	}

	return fmt.Errorf("error")
}

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
	if id == "" || glob.DS == nil {
		return ""
	}
	g := glob.Guild

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

	if id == "" || glob.DS == nil {
		return ""
	}
	g := glob.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				return m.User.AvatarURL(fmt.Sprintf("%v", size))
			}
		}
	}

	return ""
}

func GetDiscordIDFromFactorioName(pname string) string {

	if pname == "" {
		return ""
	}

	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()

	if glob.PlayerList[pname] != nil {
		return glob.PlayerList[pname].ID
	}
	return ""
}

func GetFactorioNameFromDiscordID(id string) string {

	if id == "" {
		return ""
	}

	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()

	for i, player := range glob.PlayerList {
		if glob.PlayerList[i].ID == id {
			return player.Name
		}
	}
	return ""
}
