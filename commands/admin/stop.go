package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/*  StopServer saves the map and closes Factorio.  */
func StopServer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fact.SetRelaunchThrottle(0)
	fact.SetAutoStart(false)
	if fact.IsFactRunning() {

		fact.CMS(cfg.Local.Channel.ChatChannel, "Stopping Factorio, and disabling auto-launch.")
		fact.QuitFactorio()
	} else {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio isn't running, disabling auto-launch")
	}

}
