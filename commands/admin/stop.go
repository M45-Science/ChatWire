package admin

import (
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/bwmarrin/discordgo"
)

// StopServer saves and stops the server.
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
