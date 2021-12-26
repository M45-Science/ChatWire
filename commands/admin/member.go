package admin

import (
	"fmt"

	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

// StopServer saves and stops the server.
func SetPlayerMember(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)

	if argnum > 0 {
		for i := 0; i < argnum; i++ {
			pname := args[i]
			fact.PlayerLevelSet(pname, 1)
			fact.AutoPromote(pname)
		}

		fact.CMS(m.ChannelID, fmt.Sprintf("%d players given members status.", argnum))
	} else {
		fact.CMS(m.ChannelID, "Error! Correct syntax: member name1,name2")
	}

}
