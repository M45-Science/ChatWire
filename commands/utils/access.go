package utils

import (
	//"fmt"

	"fmt"
	"math/rand"
	"time"

	"../../support"
	"github.com/bwmarrin/discordgo"
	//b64 "encoding/base64"
)

func AccessServer(s *discordgo.Session, m *discordgo.MessageCreate) {

	rand.Seed(time.Now().UnixNano())
	all := "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789"
	length := 16
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = all[rand.Intn(len(all))]
	}
	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})
	str := string(buf)

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, fmt.Sprintf("Access Code: `%s`\nType /access `%s` on any of our factorio servers to be verified.", str, str))
	if err != nil {
		support.ErrorLog(err)
	}
	return
}
