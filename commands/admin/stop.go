package admin

import (
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

// StopServer saves the map and closes Factorio.
func StopServer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	fact.SetRelaunchThrottle(0)
	fact.SetAutoStart(false)
	if fact.IsFactRunning() {

		fact.CMS(m.ChannelID, "Stopping Factorio, and disabling auto-launch.")
		fact.QuitFactorio()
	} else {
		fact.CMS(m.ChannelID, "Factorio isn't running, disabling auto-launch")
	}

}
