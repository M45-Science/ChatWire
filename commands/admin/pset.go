package admin

import (
	"fmt"
	"strconv"

	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/bwmarrin/discordgo"
)

func SetPlayerLevel(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)

	if argnum > 1 {
		pname := args[0]
		plevel, _ := strconv.Atoi(args[1])
		fact.PlayerLevelSet(pname, plevel)
		fact.CMS(m.ChannelID, fmt.Sprintf("Set: Player: %s, Level: %d", pname, plevel))
	} else {
		fact.CMS(m.ChannelID, "Error! Correct syntax: playername level")
	}

}
