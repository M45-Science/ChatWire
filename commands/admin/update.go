package admin

import (
	"strings"

	"github.com/Distortions81/M45-ChatWire/cfg"
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/bwmarrin/discordgo"
)

func Update(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	argnum := len(args)

	if cfg.Global.PathData.FactUpdaterPath != "" {
		if argnum > 0 && strings.ToLower(args[0]) == "cancel" {
			fact.SetDoUpdateFactorio(false)
			cfg.Local.AutoUpdate = false
			fact.CMS(m.ChannelID, "Update canceled, and update check disabled.")
			return
		}
		fact.CheckFactUpdate(true)
	} else {
		fact.CMS(m.ChannelID, "AutoUpdate is disabled.")
	}
}
