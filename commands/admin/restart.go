package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboots Factorio only */
func Restart(s *discordgo.Session, i *discordgo.InteractionCreate) {

	fact.CMS(cfg.Local.Channel.ChatChannel, "Now starting!")

	fact.SetAutoStart(true)
	fact.SetRelaunchThrottle(0)
	if fact.IsFactRunning() {
		fact.QuitFactorio()
	}
}
