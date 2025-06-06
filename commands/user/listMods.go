package user

import (
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/modupdate"

	"github.com/bwmarrin/discordgo"
)

func ListGameMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}

	modFiles, err := modupdate.GetModFiles()
	if err != nil {
		disc.InteractionEphemeralResponse(i, "List-Mods", "Unable to read mod files.")
		return
	}
	modList, _ := modupdate.GetModList()
	mergedMods := modupdate.MergeModLists(modFiles, modList)

	buf := ""
	for _, item := range mergedMods {
		if item.Name == "base" {
			continue
		}
		if !item.Enabled {
			continue
		}
		if buf != "" {
			buf = buf + "\n"
		}

		if modupdate.IsBaseMod(item.Name) {
			buf = buf + item.Name + " (base mod)"
		} else if item.Version != constants.Unknown {
			buf = buf + item.Name + " (" + item.Version + ")"
		} else {
			buf = buf + item.Name
		}
	}

	if buf == "" {
		buf = "No mods installed."
	}

	disc.InteractionEphemeralResponse(i, "List-Mods", buf)
}
