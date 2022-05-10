package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func DebugStat(s *discordgo.Session, i *discordgo.InteractionCreate) {

	glob.PlayerSusLock.Lock()
	var buf string = "Debug:\nSusList:"
	for pname, score := range glob.PlayerSus {
		if glob.PlayerSus[pname] > 0 {
			buf = buf + fmt.Sprintf("%v: %v\n", pname, score)
		}
	}
	fact.CMS(cfg.Local.Channel.ChatChannel, buf)
	glob.PlayerSusLock.Unlock()

	glob.ChatterLock.Lock()
	buf = "\nChatterList:"
	for pname, score := range glob.ChatterSpamScore {
		if glob.PlayerSus[pname] > 0 {
			buf = buf + fmt.Sprintf("%v: %v\n", pname, score)
		}
	}
	fact.CMS(cfg.Local.Channel.ChatChannel, buf)
	glob.ChatterLock.Unlock()
}
