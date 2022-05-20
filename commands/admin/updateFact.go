package admin

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
)

/* Update Factorio  */
func UpdateFact(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split("", " ")
	argnum := len(args)

	if cfg.Global.Paths.Binaries.FactUpdater != "" {
		if argnum > 0 && strings.ToLower(args[0]) == "cancel" {
			fact.SetDoUpdateFactorio(false)
			cfg.Local.Options.AutoUpdate = false

			buf := "Update canceled, and auto-update disabled."
			disc.EphemeralResponse(s, i, "Status:", buf)
			return
		}
		fact.CheckFactUpdate(true)
	} else {
		buf := "The Factorio updater isn't configured."
		disc.EphemeralResponse(s, i, "Error:", buf)
	}
}
