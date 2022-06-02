package moderator

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/fact"
	"ChatWire/glob"
)

/* Load a different save-game */
func ChangeMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	glob.VoteBoxLock.Lock()
	glob.VoteBox.LastMapChange = time.Now()
	fact.VoidAllVotes() /* Void all votes */
	fact.WriteVotes()   /* Save to file before exiting */
	glob.VoteBoxLock.Unlock()

	fact.ShowMapList(s, i, false)
}
