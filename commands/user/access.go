package user

import (
	"time"

	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

/* AccessServer locks PasswordListLock
 * This allows players to register, for discord roles and in-game perks */
func AccessServer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	if !fact.IsFactRunning() {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Factorio isn't currently running.")
		return
	}
	/* Do before lock */
	g := xkcdpwgen.NewGenerator()
	g.SetNumWords(3)
	g.SetCapitalize(false)
	g.SetDelimiter("-")
	password := g.GeneratePasswordString()

	t := time.Now()

	glob.PasswordListLock.Lock()
	if glob.PassList[m.Author.ID] != nil {
		delete(glob.PassList, m.Author.ID)
		cwlog.DoLogCW("Invalidating previous unused password...")
	}
	np := glob.PassData{
		Code:   password,
		DiscID: m.Author.ID,
		Time:   t.Unix(),
	}
	glob.PassList[m.Author.ID] = &np

	glob.PasswordListLock.Unlock()

	//Rewrite reply
	fact.CMS(m.ChannelID, "Access code was direct-messaged to you (check if dms are on)!")
}
