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
	_, err := s.ChannelMessageSend(dmChannel.ID, fmt.Sprintf("Access Code: `%s`\n \nPress ~ or to \\` to access chat,\nthen type `/access %s`, and press enter.\nYou can do this on any of our factorio servers to be verified!\nYou can copy-paste (control-c, control-v) the code from discord and into factorio.", password, password))
	if err != nil {
		support.ErrorLog(err)
	}
	return
}
