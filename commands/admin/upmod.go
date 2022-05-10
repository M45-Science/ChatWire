package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/modupdate"

	"github.com/bwmarrin/discordgo"
)

func ForceUpdateMods(s *discordgo.Session, i *discordgo.InteractionCreate) {

	modupdate.CheckMods(true)
	fact.CMS(cfg.Local.Channel.ChatChannel, "Updating mods.")
}
