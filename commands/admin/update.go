package admin

import (
	"strings"

	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Update Factorio  */
func Update(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split("", " ")
	argnum := len(args)

	if cfg.Global.Paths.Binaries.FactUpdater != "" {
		if argnum > 0 && strings.ToLower(args[0]) == "cancel" {
			fact.SetDoUpdateFactorio(false)
			cfg.Local.Options.AutoUpdate = false
			fact.CMS(cfg.Local.Channel.ChatChannel, "Update canceled, and update check disabled.")
			return
		}
		fact.CheckFactUpdate(true)
	} else {
		fact.CMS(cfg.Local.Channel.ChatChannel, "AutoUpdate is disabled.")
	}
}
