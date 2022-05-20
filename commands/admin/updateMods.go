package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/modupdate"
)

func UpdateMods(s *discordgo.Session, i *discordgo.InteractionCreate) {

	embed := &discordgo.MessageEmbed{Title: "Status:", Description: "Attempting to update game mods.\n"}
	disc.InteractionResponse(s, i, embed)
	modupdate.CheckMods(true, true)
}
