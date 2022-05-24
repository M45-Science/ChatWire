package moderator

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* Load a different save-game */
func RewindMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	a := i.ApplicationCommandData()
	var autosaveNum int64 = -1

	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionInteger {
			autosaveNum = arg.IntValue()
		} else if arg.Type == discordgo.ApplicationCommandOptionBoolean {
			if arg.BoolValue() {
				fact.ShowRewindList(s, i)
				return
			}
		}
	}

	/* Correct number of arguments (1) */
	if autosaveNum > 0 && autosaveNum <= int64(cfg.Global.Options.AutosaveMax) {

		glob.VoteBoxLock.Lock()
		glob.VoteBox.LastRewindTime = time.Now()
		fact.VoidAllVotes()     /* Void all votes */
		fact.WriteRewindVotes() /* Save to file before exiting */
		glob.VoteBoxLock.Unlock()

		fact.DoRewindMap(s, fmt.Sprintf("%v", autosaveNum))
		return
	}

	fact.ShowRewindList(s, i)

}
