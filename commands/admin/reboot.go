package admin

import (
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/bwmarrin/discordgo"
)

// Restart saves and restarts the server
func Reboot(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	fact.CMS(m.ChannelID, "Now rebooting!")
	fact.SetRelaunchThrottle(0)
	fact.DoExit()
}
