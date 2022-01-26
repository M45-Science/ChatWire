package admin

import (
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/*  Restart saves and restarts the server */
func Reload(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	fact.CMS(m.ChannelID, "Now reloading!")

	fact.SetCWReboot(true)
	fact.SetRelaunchThrottle(0)
	fact.QuitFactorio()
}
