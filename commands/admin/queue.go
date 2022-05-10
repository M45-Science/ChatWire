package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboot when server is empty */
func Queue(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsQueued() {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Reload is now queued.")
		fact.SetQueued(true)
	}
}
