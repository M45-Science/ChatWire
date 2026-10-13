package modupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
