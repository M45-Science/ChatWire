package user

import (
	"errors"
	"os"
	"sort"
	"strings"

	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/modedit"
	"ChatWire/modupdate"

	"github.com/bwmarrin/discordgo"
)

func ListGameMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}
	buf, err := gameModListText()
	if err != nil {
		disc.InteractionEphemeralResponse(i, "List-Mods: Error", err.Error())
		return
	}
	if len(buf) > constants.MaxDiscordMsgLen {
		disc.InteractionEphemeralFileResponse(i, "List-Mods", "See attached mod list.", "mod-list.txt", []byte(buf))
	} else {
		disc.InteractionEphemeralResponse(i, "List-Mods", buf)
	}
}

func gameModListText() (string, error) {
	modFiles, err := modupdate.GetModFiles()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	builtins, err := modupdate.GetBuiltinModFiles()
	if err != nil {
		return "", err
	}
	modList, err := modupdate.GetModList()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	mergedMods := modupdate.MergeModLists(append(modFiles, builtins...), modList)
	sort.Slice(mergedMods, func(i, j int) bool { return strings.ToLower(mergedMods[i].Name) < strings.ToLower(mergedMods[j].Name) })
	configured := map[string]bool{}
	for _, mod := range modList.Mods {
		configured[mod.Name] = true
	}
	prefs := modedit.ReadPrefs()
	prefMap := map[string]string{}
	for _, p := range prefs.Mods {
		prefMap[strings.ToLower(p.Name)] = p.Version
	}

	var enabled, disabled, unconfigured []string
	for _, item := range mergedMods {
		line := item.Name
		if item.Version != "" && item.Version != "0.0.0" && item.Version != constants.Unknown {
			line += " (" + item.Version + ")"
		}
		if modupdate.IsBaseMod(item.Name) {
			line += " (built-in)"
			if !configured[item.Name] {
				unconfigured = append(unconfigured, line)
				continue
			}
		} else if v := prefMap[strings.ToLower(item.Name)]; v != "" && !strings.EqualFold(v, "auto") {
			line += " (force " + v + ")"
		}
		if item.Enabled {
			enabled = append(enabled, line)
		} else {
			disabled = append(disabled, line)
		}
	}
	section := func(name string, lines []string) string {
		if len(lines) == 0 {
			return name + ": None"
		}
		return name + ":\n" + strings.Join(lines, "\n")
	}
	sections := []string{section("Enabled", enabled), section("Disabled", disabled)}
	if len(unconfigured) > 0 {
		sections = append(sections, section("Built-in mods: not configured", unconfigured))
	}
	return strings.Join(sections, "\n\n"), nil
}
