package utils

import (
	"fmt"
	//"math/rand"
	//"time"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

func AccessServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	if glob.PasswordPos >= glob.MaxPasswords {
		glob.PasswordPos = 0
	}

	g := xkcdpwgen.NewGenerator()
	g.SetNumWords(5)
	g.SetCapitalize(false)
	g.SetDelimiter("-")
	password := g.GeneratePasswordString()

	glob.PasswordList[glob.PasswordPos] = password
	glob.PasswordName[glob.PasswordPos] = m.Author.Username
	glob.PasswordPos++

	dmChannel, _ := glob.DS.UserChannelCreate(m.Author.ID)
	_, err := s.ChannelMessageSend(dmChannel.ID, fmt.Sprintf("Access Code: `%s`\n \nIn factorio, press ~ or to \\` to access chat,\nthen paste or type `/access %s`, and press enter.\nYou can do this on any of our factorio servers to be verified!\nYou can copy-paste (control-c, control-v) the code from Discord and into Factorio.\n \nTHIS PASSWORD ONLY WORKS ONCE, SO DON'T SHARE IT!", password, password))
	if err != nil {
		support.ErrorLog(err)
	}

	_, errb := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Access code sent privately.")
	if errb != nil {
		support.ErrorLog(errb)
	}
	return
}
