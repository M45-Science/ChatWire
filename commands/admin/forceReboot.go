package admin

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
)

/* Reboots cw */
func ForceReboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	buf := "Rebooting!"
	embed := &discordgo.MessageEmbed{Title: "Status:", Description: buf}
	disc.InteractionResponse(s, i, embed)
	fact.SetRelaunchThrottle(0)
	fact.DoExit(false)
}
