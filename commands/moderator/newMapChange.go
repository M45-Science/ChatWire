package moderator

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/fact"
	"ChatWire/glob"
)

/* Load a different save-game */
func ChangeMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var saveName string
	var list bool

	a := i.ApplicationCommandData()

	//Get args
	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			saveName = arg.StringValue()
		} else if arg.Type == discordgo.ApplicationCommandOptionBoolean {
			list = arg.BoolValue()
		}
	}

	if saveName != "" {
		num, err := strconv.Atoi(saveName)
		//Expand name to autosave if only a number was supplied
		if err == nil && num > 0 {
			saveName = fmt.Sprintf("_autosave%v", num)
		}
		fact.DoChangeMap(s, saveName)
		return
	} else if list {
		fact.ShowFullMapList(s, i)
		return
	}

	glob.VoteBoxLock.Lock()
	glob.VoteBox.LastMapChange = time.Now()
	fact.VoidAllVotes() /* Void all votes */
	fact.WriteVotes()   /* Save to file before exiting */
	glob.VoteBoxLock.Unlock()

	fact.ShowMapList(s, i, false)
}
