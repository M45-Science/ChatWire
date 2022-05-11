package admin

import (
	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* Reboots cw */
func Reboot(s *discordgo.Session, i *discordgo.InteractionCreate) {

	buf := "Rebooting!"
	embed := &discordgo.MessageEmbed{Title: "Status:", Description: buf}
	disc.InteractionResponse(s, i, embed)
	fact.SetRelaunchThrottle(0)
	fact.DoExit(false)
}
