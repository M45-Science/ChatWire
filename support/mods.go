package support

import (
	"os"
	"strings"

	"ChatWire/cfg"
)

func GetModFiles() []string {
	modPath := cfg.GetModsFolder()

	modList, errm := os.ReadDir(modPath)
	modStrings := []string{}

	if errm == nil {
		for _, mod := range modList {
			if strings.HasSuffix(mod.Name(), ".zip") {
				modStrings = append(modStrings, strings.TrimSuffix(mod.Name(), ".zip"))
			}
		}
	}

	return modStrings
}
