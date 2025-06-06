package modedit

import (
	"encoding/json"
	"os"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
)

type VersionPrefs struct {
	Mods []ModVersion `json:"mods"`
}

type ModVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func versionFilePath() string {
	return cfg.GetModsFolder() + constants.ModVersionsName
}

func ReadPrefs() VersionPrefs {
	path := versionFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return VersionPrefs{}
	}
	out := VersionPrefs{}
	if json.Unmarshal(data, &out) != nil {
		return VersionPrefs{}
	}
	return out
}

func WritePrefs(prefs VersionPrefs) error {
	path := versionFilePath()
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func GetVersion(prefs VersionPrefs, name string) string {
	for _, m := range prefs.Mods {
		if strings.EqualFold(m.Name, name) {
			return m.Version
		}
	}
	return ""
}

func SetVersion(name, version string) error {
	prefs := ReadPrefs()
	idx := -1
	for i, m := range prefs.Mods {
		if strings.EqualFold(m.Name, name) {
			idx = i
			break
		}
	}
	if idx >= 0 {
		prefs.Mods = append(prefs.Mods[:idx], prefs.Mods[idx+1:]...)
	}
	if version != "" {
		prefs.Mods = append(prefs.Mods, ModVersion{Name: name, Version: version})
	}
	return WritePrefs(prefs)
}
