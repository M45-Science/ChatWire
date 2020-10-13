package user

import (
	"fmt"

	"../../constants"
	"../../fact"
	"../../glob"

	"github.com/bwmarrin/discordgo"
)

// online executes /p o on the server.
func GameVersion(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	buf := fmt.Sprintf("Factorio version: v%s\nBot version: v%s", glob.FactorioVersion, constants.Version)

	fact.CMS(m.ChannelID, buf)
}
