package admin

import (
	"../../fact"
	"github.com/bwmarrin/discordgo"
)

// Restart saves and restarts the server
func Restart(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	fact.CMS(m.ChannelID, "Now starting!")

	fact.SetAutoStart(true)
	fact.SetRelaunchThrottle(0)
	if fact.IsFactRunning() {
		fact.QuitFactorio()
	}

	return
}
