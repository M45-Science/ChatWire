package admin

import (
	"fmt"

	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

// StopServer saves and stops the server.
func SetPlayerRegular(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)

	if argnum > 0 {
		for i := 0; i < argnum; i++ {
			pname := args[i]
			fact.PlayerLevelSet(pname, 2)
			fact.AutoPromote(pname)
		}

		fact.CMS(m.ChannelID, fmt.Sprintf("%d players given regulars status.", argnum))
	} else {
		fact.CMS(m.ChannelID, "Error! Correct syntax: regular name1,name2")
	}

}
