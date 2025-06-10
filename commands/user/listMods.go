package user

import (
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/modedit"
	"ChatWire/modupdate"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func ListGameMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}

	modFiles, err := modupdate.GetModFiles()
	if err != nil {
		disc.InteractionEphemeralResponse(i, "List-Mods: Error", "Unable to read mod files.")
		return
	}
	modList, _ := modupdate.GetModList()
	mergedMods := modupdate.MergeModLists(modFiles, modList)

	prefs := modedit.ReadPrefs()
	prefMap := map[string]string{}
	for _, p := range prefs.Mods {
		prefMap[strings.ToLower(p.Name)] = p.Version
	}

	ebuf := ""
	for _, item := range mergedMods {
		if item.Name == "base" || !item.Enabled {
			continue
		}
		if ebuf != "" {
			ebuf += "\n"
		}
		line := ""
		if modupdate.IsBaseMod(item.Name) {
			line = item.Name + " (base mod)"
		} else if item.Version != "" {
			ebuf = ebuf + item.Name + " (" + item.Version + ")"
		} else {
			line = item.Name
		}
		if v, ok := prefMap[strings.ToLower(item.Name)]; ok && v != "" {
			line += " **(FORCE " + v + ")**"
		}
		ebuf += line
	}

	dbuf := ""
	for _, item := range mergedMods {
		if item.Name == "base" || item.Enabled {
			continue
		}
		if dbuf != "" {
			dbuf += "\n"
		}
		line := ""
		if modupdate.IsBaseMod(item.Name) {
			line = item.Name + " (base mod)"
		} else if item.Version != constants.Unknown {
			line = item.Name + " (" + item.Version + ")"
		} else {
			line = item.Name
		}
		if v, ok := prefMap[strings.ToLower(item.Name)]; ok && v != "" {
			line += " (force " + v + ")"
		}
		dbuf += line
	}

	if ebuf == "" {
		ebuf = "Enabled: None"
	} else {
		ebuf = "Enabled:\n" + ebuf
	}

	if dbuf == "" {
		dbuf = "\n\nDisabled: None"
	} else {
		dbuf = "Disabled:\n" + dbuf
		if ebuf != "" {
			dbuf = "\n\n" + dbuf
		}
	}

	buf := ebuf + dbuf
	if buf == "" {
		buf = "No mods installed."
	}

	disc.InteractionEphemeralResponse(i, "List-Mods", buf)
}
