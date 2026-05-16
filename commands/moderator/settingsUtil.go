package moderator

import (
	"os"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
)

/* Get list of map generation presets, because an invalid one will make map generation fail */
func getMapGenNames() []string {
	output := []string{"none", constants.CustomMapGeneratorName}

	path := cfg.GetSharedMapGeneratorFolder()
	files, err := os.ReadDir(path)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return output
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), "-gen.json") {
			name := strings.TrimSuffix(f.Name(), "-gen.json")
			if !strings.EqualFold(name, constants.CustomMapGeneratorName) {
				output = append(output, name)
			}
		}
	}
	return output
}

/* See if this map gen exists */
func checkMapGen(text string) bool {

	/* Allow no generator */
	if text == "" || text == "none" {
		return true
	}
	genNames := getMapGenNames()
	for _, name := range genNames {
		if strings.EqualFold(name, text) {
			return true
		}
	}
	return false
}

func checkMapTypes(text string) bool {

	names := constants.MapTypes
	for _, name := range names {
		if strings.EqualFold(name, text) {
			return true
		}
	}
	return false
}
