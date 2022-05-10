package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboots cw */
func Reboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	fact.CMS(cfg.Local.Channel.ChatChannel, "Now rebooting!")
	fact.SetRelaunchThrottle(0)
	fact.DoExit(false)
}
