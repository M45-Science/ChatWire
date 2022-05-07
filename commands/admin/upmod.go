package admin

import (
	"ChatWire/fact"
	"ChatWire/modupdate"

	"github.com/bwmarrin/discordgo"
)

func ForceUpdateMods(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	modupdate.CheckMods(true)
	fact.CMS(m.ChannelID, "Updating mods.")
}
