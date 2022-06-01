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
func ChangeMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var autosaveNum int64 = -1

	/* Correct number of arguments (1) */
	if autosaveNum > 0 && autosaveNum <= int64(cfg.Global.Options.AutosaveMax) {

		glob.VoteBoxLock.Lock()
		glob.VoteBox.LastMapChange = time.Now()
		fact.VoidAllVotes() /* Void all votes */
		fact.WriteVotes()   /* Save to file before exiting */
		glob.VoteBoxLock.Unlock()

		fact.DoChangeMap(s, fmt.Sprintf("%v", autosaveNum))
		return
	}

	fact.ShowMapList(s, i, false)
}
