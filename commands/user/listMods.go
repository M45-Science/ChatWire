package user

import (
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/support"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func ListGameMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}

	buf := ""
	modList, mErr := modupdate.GetModList()
	if mErr == nil {
		for _, mod := range modList.Mods {
			if modupdate.IsBaseMod(mod.Name) {
				continue
			}
			if !mod.Enabled {
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

	if buf == "" {
		buf = strings.Join(support.GetModFiles(), ", ")
	}

	disc.InteractionEphemeralResponse(i, "Mod files:", buf)
}
