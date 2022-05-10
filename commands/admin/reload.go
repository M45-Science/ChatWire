package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/*  Restart saves and restarts the server */
func Reload(s *discordgo.Session, i *discordgo.InteractionCreate) {

	fact.CMS(cfg.Local.Channel.ChatChannel, "Now reloading!")

	fact.SetCWReboot(true)
	fact.SetRelaunchThrottle(0)
	fact.QuitFactorio()
}
