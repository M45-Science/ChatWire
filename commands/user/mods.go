package user

import (
	"../../constants"
	"../../fact"
	"../../glob"

	"github.com/bwmarrin/discordgo"
)

//MosList locks ModLoadLock (READ)
func ModsList(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	if !fact.IsFactRunning() {
		fact.CMS(m.ChannelID, "Factorio is not running.")
		return
	}
	if !fact.IsFactorioBooted() {
		fact.CMS(m.ChannelID, "Factorio not done loading yet.")
		return
	}

	glob.ModLoadLock.RLock()
	defer glob.ModLoadLock.RUnlock()

	if glob.ModLoadString == constants.Unknown {
		fact.CMS(m.ChannelID, "No mods loaded.")
	} else {
		fact.CMS(m.ChannelID, "Mod list: "+glob.ModLoadString)
	}
}
