package modupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ChatWire/cfg"
)

// GetBuiltinModFiles inventories bundled mods for display. Keep them out of
// GetModFiles, whose archive paths are used for portal downloads and modpacks.
func GetBuiltinModFiles() ([]modZipInfo, error) {
	directory := filepath.Join(cfg.GetFactorioFolder(), "data")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var mods []modZipInfo
	for _, entry := range entries {
		if !IsBaseMod(entry.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name(), "info.json"))
		if err != nil {
			return nil, fmt.Errorf("read built-in mod %s: %w", entry.Name(), err)
		}
		var mod modZipInfo
		if err := json.Unmarshal(data, &mod); err != nil {
			return nil, fmt.Errorf("read built-in mod %s: %w", entry.Name(), err)
		}
		mod.Name = entry.Name()
		mods = append(mods, mod)
	}
	return mods, nil
}

// ModInventoryEntry separates explicit configuration from bundled defaults.
type ModInventoryEntry struct {
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	Version      string   `json:"version"`
	Enabled      bool     `json:"enabled"`
	Configured   bool     `json:"configured"`
	Builtin      bool     `json:"builtin"`
	Dependencies []string `json:"dependencies,omitempty"`
}

func GetModInventory() ([]ModInventoryEntry, error) {
	list, err := GetModList()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	builtins, err := GetBuiltinModFiles()
	if err != nil {
		return nil, err
	}
	configured := map[string]ModData{}
	for _, mod := range list.Mods {
		configured[mod.Name] = mod
	}
	out := []ModInventoryEntry{}
	for _, mod := range builtins {
		entry, exists := configured[mod.Name]
		out = append(out, ModInventoryEntry{Name: mod.Name, Title: mod.Title, Version: mod.Version, Enabled: entry.Enabled, Configured: exists, Builtin: true, Dependencies: mod.Dependencies})
		delete(configured, mod.Name)
	}
	for _, mod := range configured {
		out = append(out, ModInventoryEntry{Name: mod.Name, Enabled: mod.Enabled, Configured: true, Builtin: IsBaseMod(mod.Name)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Builtin != out[j].Builtin {
			return out[i].Builtin
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// SetBuiltinModEnabled enables required bundled dependencies from the installed
// release, including Recycler in 2.1. Explicitly disabled recommendations remain
// disabled, matching Factorio's '+' dependency semantics.
func SetBuiltinModEnabled(list ModListData, name string, enabled bool) (ModListData, error) {
	mods, err := GetBuiltinModFiles()
	if err != nil {
		return list, err
	}
	installed := map[string]modZipInfo{}
	for _, mod := range mods {
		installed[mod.Name] = mod
	}
	if _, ok := installed[name]; !ok {
		return list, fmt.Errorf("built-in mod %s is not installed", name)
	}
	if name == "base" && !enabled {
		return list, errors.New("the base mod is required")
	}
	list.Mods = append([]ModData(nil), list.Mods...)
	seen := map[string]bool{}
	var set func(string, bool) error
	set = func(name string, value bool) error {
		if seen[name] {
			return nil
		}
		seen[name] = true
		idx := -1
		for i, mod := range list.Mods {
			if mod.Name == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			list.Mods = append(list.Mods, ModData{Name: name, Enabled: value})
		} else {
			list.Mods[idx].Enabled = value
		}
		if !value {
			return nil
		}
		for _, raw := range installed[name].Dependencies {
			recommended := strings.HasPrefix(strings.TrimSpace(raw), "+")
			dep := parseDep(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "+")))
			if dep.optional || dep.incompatible {
				continue
			}
			if recommended {
				skip := false
				for _, mod := range list.Mods {
					if mod.Name == dep.name && !mod.Enabled {
						skip = true
					}
				}
				if skip {
					continue
				}
			}
			if _, ok := installed[dep.name]; !ok {
				return fmt.Errorf("%s requires unavailable built-in mod %s", name, dep.name)
			}
			if err := set(dep.name, true); err != nil {
				return err
			}
		}
		return nil
	}
	err = set(name, enabled)
	return list, err
}

func ValidateBuiltinModDependencies(list ModListData) error {
	mods, err := GetBuiltinModFiles()
	if err != nil {
		return err
	}
	installed := map[string]modZipInfo{}
	enabled := map[string]bool{}
	for _, mod := range mods {
		installed[mod.Name] = mod
		enabled[mod.Name] = true
	}
	for _, mod := range list.Mods {
		enabled[mod.Name] = mod.Enabled
	}
	for _, mod := range mods {
		if !enabled[mod.Name] {
			continue
		}
		for _, raw := range mod.Dependencies {
			if strings.HasPrefix(strings.TrimSpace(raw), "+") {
				continue
			}
			dep := parseDep(raw)
			if dep.optional || dep.incompatible {
				continue
			}
			actual, ok := installed[dep.name]
			if !ok || !enabled[dep.name] {
				return fmt.Errorf("%s requires enabled built-in mod %s", mod.Name, dep.name)
			}
			if dep.version != "" {
				good, err := checkVersion(dep.equality, dep.version, actual.Version)
				if err != nil || !good {
					return fmt.Errorf("%s requires %s", mod.Name, raw)
				}
			}
		}
	}
	return nil
}
