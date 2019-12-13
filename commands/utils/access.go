package utils

import (
	//"fmt"

	"../../glob"
	"../../support"
	"github.com/Distortions81/discordgo"
	//b64 "encoding/base64"
)

func AccessServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	if len(glob.CharName) < 5 {
		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Name too short...\n")
		if err != nil {
			support.ErrorLog(err)
		}
		return
	}
	if len(glob.CharName) < 64 {
		_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Name too long...\n")
		if err != nil {
			support.ErrorLog(err)
		}
		return
	}

	//_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, fmt.Sprintf("Access Code: %s\n", m) )
	//if err != nil {
	//	support.ErrorLog(err)
	//}
	return
}
