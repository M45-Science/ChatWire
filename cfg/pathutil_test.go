package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"ChatWire/constants"
)

func TestGetMapGeneratorFilesPrefersDirectoryLayout(t *testing.T) {
	oldGlobal := Global
	t.Cleanup(func() {
		Global = oldGlobal
	})

	tmp := t.TempDir()
	Global.Paths.Folders.ServersRoot = tmp
	Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	dir := filepath.Join(tmp, constants.DefaultMapGeneratorsDir, "spiral")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed creating generator dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, constants.MapGenSettingsName), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing map-gen settings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, constants.MapSettingsName), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing map settings: %v", err)
	}

	genPath, setPath := GetMapGeneratorFiles("spiral")
	if genPath != filepath.Join(dir, constants.MapGenSettingsName) {
		t.Fatalf("unexpected map-gen path: %q", genPath)
	}
	if setPath != filepath.Join(dir, constants.MapSettingsName) {
		t.Fatalf("unexpected map-settings path: %q", setPath)
	}
}

func TestGetMapGeneratorFilesFallsBackToLegacyLayout(t *testing.T) {
	oldGlobal := Global
	t.Cleanup(func() {
		Global = oldGlobal
	})

	tmp := t.TempDir()
	Global.Paths.Folders.ServersRoot = tmp
	Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	genPath, setPath := GetMapGeneratorFiles("legacy")
	if genPath != filepath.Join(tmp, constants.DefaultMapGeneratorsDir, "legacy-gen.json") {
		t.Fatalf("unexpected legacy map-gen path: %q", genPath)
	}
	if setPath != filepath.Join(tmp, constants.DefaultMapGeneratorsDir, "legacy-set.json") {
		t.Fatalf("unexpected legacy map-settings path: %q", setPath)
	}
}
