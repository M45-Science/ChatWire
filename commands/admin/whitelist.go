package admin

import (
	"fmt"

	"../../cfg"
	"../../fact"
	"github.com/bwmarrin/discordgo"
)

//WriteWhitelist locks PlayerListLock (READ)
func WriteWhitelist(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	//Write whitelist
	if cfg.Local.SoftModOptions.DoWhitelist {
		count := fact.WriteWhitelist()
		if count > 0 {
			fact.CMS(m.ChannelID, fmt.Sprintf("Wrote whitelist of %x players", count))
		} else {
			fact.CMS(m.ChannelID, "Empty whitelist written")
		}
	} else {
		fact.CMS(m.ChannelID, "whitelist isn't enabled.")
	}

}
