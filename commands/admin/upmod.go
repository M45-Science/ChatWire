package admin

import (
	"ChatWire/disc"
	"ChatWire/modupdate"

	"github.com/bwmarrin/discordgo"
)

func ForceUpdateMods(s *discordgo.Session, i *discordgo.InteractionCreate) {

	embed := &discordgo.MessageEmbed{Title: "Status:", Description: "Attempting to update game mods.\n"}
	disc.InteractionResponse(s, i, embed)
	modupdate.CheckMods(true, true)
}
