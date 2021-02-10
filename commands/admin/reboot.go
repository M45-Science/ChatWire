package admin

import (
	"../../fact"
	"github.com/bwmarrin/discordgo"
)

// Restart saves and restarts the server
func Reboot(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	fact.CMS(m.ChannelID, "Now rebooting!")
	fact.SetRelaunchThrottle(0)
	fact.DoExit()
}
