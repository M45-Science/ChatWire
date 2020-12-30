package admin

import (
	"../../config"
	"../../fact"
	"../../glob"
	"github.com/bwmarrin/discordgo"
)

//Archive map
func ReloadConfig(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	glob.GameMapLock.Lock()
	defer glob.GameMapLock.Unlock()

	config.Config.LoadEnv()
	fact.CMS(m.ChannelID, "Configuration refreshed.")

}
