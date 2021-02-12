package admin

import (
	"../../fact"
	"../../glob"
	"github.com/bwmarrin/discordgo"
)

//Archive map
func ReloadConfig(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	glob.GameMapLock.Lock()
	defer glob.GameMapLock.Unlock()

	fact.CMS(m.ChannelID, "Not yet implemented")

}
