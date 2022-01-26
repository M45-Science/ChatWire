package admin

import (
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboot when server is empty */
func Queue(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	if !fact.IsQueued() {
		fact.CMS(m.ChannelID, "Reload is now queued.")
		fact.SetQueued(true)
	}
}
