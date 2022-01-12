package admin

import (
	"ChatWire/fact"
	"ChatWire/glob"
	"time"

	"github.com/bwmarrin/discordgo"
)

//Load a different save-game
func Rewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)

	//Correct number of arguments (1)
	if argnum == 1 {

		glob.VoteBoxLock.Lock()
		glob.VoteBox.LastRewindTime = time.Now()
		fact.VoidAllVotes()     //Void all votes
		fact.WriteRewindVotes() //Save to file before exiting
		glob.VoteBoxLock.Unlock()

		fact.DoRewindMap(s, m, args[0])
	} else {
		fact.ShowRewindList(s, m)
		return
	}
}
