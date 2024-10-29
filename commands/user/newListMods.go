package user

import (
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/support"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func ListGameMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}

	buf := ""
	modList, mErr := support.GetGameMods()
	if mErr == nil && modList != nil {
		for _, mod := range modList.Mods {
			if strings.EqualFold(mod.Name, "base") {
				continue
			}
			if buf != "" {
				buf = buf + ", "
			}
			if mod.Enabled {
				buf = buf + mod.Name
			}
		}
	}

	disc.EphemeralResponse(i, "Mod files:", buf)
}
