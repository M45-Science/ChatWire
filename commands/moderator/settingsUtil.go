package moderator

import (
	"os"
	"sort"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
)

/* Get list of map generators, because an invalid one will make map generation fail */
func getMapGenNames() []string {
	output := []string{"none"}
	if mapGeneratorFilesExist(constants.CustomMapGeneratorName) {
		output = append(output, constants.CustomMapGeneratorName)
	}

	path := cfg.GetSharedMapGeneratorFolder()
	files, err := os.ReadDir(path)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return output
	}

	startSort := len(output)
	found := map[string]bool{}
	for _, f := range files {
		name := ""
		if f.IsDir() {
			name = f.Name()
		} else if strings.HasSuffix(f.Name(), "-gen.json") {
			name = strings.TrimSuffix(f.Name(), "-gen.json")
		}

		key := strings.ToLower(name)
		if name != "" && !strings.EqualFold(name, constants.CustomMapGeneratorName) && !found[key] && mapGeneratorFilesExist(name) {
			output = append(output, name)
			found[key] = true
		}
	}
	sort.Strings(output[startSort:])
	return output
}

func mapGeneratorFilesExist(name string) bool {
	genPath, setPath := cfg.GetMapGeneratorFiles(name)
	if _, err := os.Stat(genPath); err != nil {
		return false
	}
	if _, err := os.Stat(setPath); err != nil {
		return false
	}
	return true
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
