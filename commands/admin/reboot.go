package admin

import (
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboots cw */
func Reboot(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	fact.CMS(m.ChannelID, "Now rebooting!")
	fact.SetRelaunchThrottle(0)
	fact.DoExit(false)
}
