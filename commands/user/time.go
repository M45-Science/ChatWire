package user

import (
	"fmt"

	"../../fact"
	"../../glob"

	"github.com/bwmarrin/discordgo"
)

//Gives play time
func GameTime(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	buf := fmt.Sprintf("Factorio play time: %s", glob.GametimeString)

	fact.CMS(m.ChannelID, buf)
}
