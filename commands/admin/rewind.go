package admin

import (
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

//Load a different save-game
func Rewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)

	//Correct number of arguments (1)
	if argnum == 1 {
		fact.DoRewindMap(s, m, args[0])
	} else {
		fact.ShowRewindList(s, m)
		return
	}
}
