package utils

import (
	//"fmt"

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
	//b64 "encoding/base64"
)

func AccessServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	if len(glob.CharName) < 5 {
		s.ChannelMessageSend(support.Config.FactorioChannelID, "Name too short...\n")
		return
	}
	if len(glob.CharName) < 64 {
		s.ChannelMessageSend(support.Config.FactorioChannelID, "Name too long...\n")
		return
	}

	//s.ChannelMessageSend(support.Config.FactorioChannelID, fmt.Sprintf("Access Code: %s\n", m) )

	return
}
