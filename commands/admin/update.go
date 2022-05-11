package admin

import (
	"strings"

	"ChatWire/cfg"
	"ChatWire/disc"
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

			buf := "Update canceled, and auto-update disabled."
			embed := &discordgo.MessageEmbed{Title: "Status:", Description: buf}
			disc.InteractionResponse(s, i, embed)
			return
		}
		fact.CheckFactUpdate(true)
	} else {
		buf := "The Factorio updater isn't configured."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
	}
}
