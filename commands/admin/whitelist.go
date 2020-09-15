package admin

import (
	"../../fact"
	"../../glob"
	"github.com/bwmarrin/discordgo"
)

//SendWhitelist locks PlayerListLock (READ)
func SendWhitelist(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	//Send whitelist
	if glob.WhitelistMode {
		glob.PlayerListLock.RLock()
		for i := 0; i <= glob.PlayerListMax; i++ {
			fact.WhitelistPlayer(glob.PlayerList[i].Name, glob.PlayerList[i].Level)
		}
		glob.PlayerListLock.RUnlock()

		fact.CMS(m.ChannelID, "Whitelist sent.")
	} else {
		fact.CMS(m.ChannelID, "whitelist isn't enabled.")
	}

}
