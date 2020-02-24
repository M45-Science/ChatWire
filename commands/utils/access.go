package utils

import (
	"fmt"
	//"math/rand"
	//"time"

	"../../support"
	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

func AccessServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	g := xkcdpwgen.NewGenerator()
	g.SetNumWords(5)
	g.SetCapitalize(true)
	password := g.GeneratePasswordString()

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, fmt.Sprintf("Access Code: `%s`\n \nPress ~ or to \` to access chat,\nthen type /access `%s`, and press enter.\nYou can do this on any of our factorio servers to be verified!\nYou can copy-paste (control-c, control-v) the code from discord and into factorio.", password, password))
	if err != nil {
		support.ErrorLog(err)
	}
	return
}
