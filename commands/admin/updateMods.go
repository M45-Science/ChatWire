package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/modupdate"
)

func UpdateMods(s *discordgo.Session, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(s, i, "Status:", "Checking for mod updates.")
	modupdate.CheckMods(true, true)
}
