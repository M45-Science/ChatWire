package admin

import (
	"strings"

	"../../config"
	"../../fact"
	"github.com/bwmarrin/discordgo"
)

func Update(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	argnum := len(args)

	if config.Config.UpdaterPath != "" {
		if argnum > 0 && strings.ToLower(args[0]) == "cancel" {
			fact.SetDoUpdateFactorio(false)
			config.Config.UpdaterPath = ""
			fact.CMS(m.ChannelID, "Update canceled, and update check disabled.")
			return
		}
		fact.CheckFactUpdate(true)
	} else {
		fact.CMS(m.ChannelID, "Updater is not configured.")
	}
}
