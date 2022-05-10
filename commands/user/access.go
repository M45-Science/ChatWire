package user

import (
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

/* AccessServer locks PasswordListLock
 * This allows players to register, for discord roles and in-game perks */
func AccessServer(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsFactRunning() {
		_, _ = s.ChannelMessageSend(cfg.Local.Channel.ChatChannel, "Factorio isn't currently running.")
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
	if glob.PassList[i.Message.Author.ID] != nil {
		delete(glob.PassList, i.Message.Author.ID)
		cwlog.DoLogCW("Invalidating previous unused password...")
	}
	np := glob.PassData{
		Code:   password,
		DiscID: i.Message.Author.ID,
		Time:   t.Unix(),
	}
	glob.PassList[i.Message.Author.ID] = &np

	glob.PasswordListLock.Unlock()

	//Rewrite reply
	fact.CMS(cfg.Local.Channel.ChatChannel, "Access code was direct-messaged to you (check if dms are on)!")
}
