package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/fact"
)

/*  Restart saves and restarts the server */
func RebootCW(s *discordgo.Session, i *discordgo.InteractionCreate) {

	fact.CMS(cfg.Local.Channel.ChatChannel, "Now reloading!")

	fact.SetCWReboot(true)
	fact.SetRelaunchThrottle(0)
	fact.QuitFactorio()
}
