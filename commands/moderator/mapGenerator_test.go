package moderator

import (
	"os"
	"path/filepath"
	"testing"

	"ChatWire/cfg"
	"ChatWire/constants"
)

func TestApplyMapGeneratorSelectionDoesNotModifyMapPreset(t *testing.T) {
	oldPreset := cfg.Local.Settings.MapPreset
	oldGenerator := cfg.Local.Settings.MapGenerator
	t.Cleanup(func() {
		cfg.Local.Settings.MapPreset = oldPreset
		cfg.Local.Settings.MapGenerator = oldGenerator
	})

	cfg.Local.Settings.MapGenerator = "custom"
	cfg.Local.Settings.MapPreset = "rail-world"

	displayName, err := applyMapGeneratorSelection("none")
	if err != nil {
		t.Fatalf("applyMapGeneratorSelection returned error: %v", err)
	}
	if displayName != "none" {
		t.Fatalf("unexpected display name: %q", displayName)
	}
	if cfg.Local.Settings.MapGenerator != "none" {
		t.Fatalf("MapGenerator = %q, want none", cfg.Local.Settings.MapGenerator)
	}
	if cfg.Local.Settings.MapPreset != "rail-world" {
		t.Fatalf("MapPreset changed to %q, want rail-world", cfg.Local.Settings.MapPreset)
	}
}

func TestSettingListDoesNotExposeMapGenerator(t *testing.T) {
	for _, setting := range SettingList {
		if setting.Name == "map-generator" {
			t.Fatal("map-generator should be changed through /map-generator, not config-server")
		}
	}
}

func TestSettingListStillExposesMapPreset(t *testing.T) {
	for _, setting := range SettingList {
		if setting.Name == "map-preset" {
			return
		}
	}
	t.Fatal("map-preset should remain a static built-in Factorio setting")
}

func TestGetMapGenNamesIncludesCompleteDirectoryLayout(t *testing.T) {
	oldGlobal := cfg.Global
	t.Cleanup(func() {
		cfg.Global = oldGlobal
	})

	tmp := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = tmp
	cfg.Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	completeDir := filepath.Join(tmp, constants.DefaultMapGeneratorsDir, "spiral")
	if err := os.MkdirAll(completeDir, 0755); err != nil {
		t.Fatalf("failed creating complete generator dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(completeDir, constants.MapGenSettingsName), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing complete generator map-gen settings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(completeDir, constants.MapSettingsName), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing complete generator map settings: %v", err)
	}

	incompleteDir := filepath.Join(tmp, constants.DefaultMapGeneratorsDir, "partial")
	if err := os.MkdirAll(incompleteDir, 0755); err != nil {
		t.Fatalf("failed creating incomplete generator dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(incompleteDir, constants.MapGenSettingsName), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing incomplete generator map-gen settings: %v", err)
	}

	names := getMapGenNames()
	if !containsString(names, "spiral") {
		t.Fatalf("expected complete directory generator in %v", names)
	}
	if containsString(names, "partial") {
		t.Fatalf("did not expect incomplete directory generator in %v", names)
	}
}

func TestApplyMapGeneratorSelectionCachesSharedGenerator(t *testing.T) {
	oldGlobal := cfg.Global
	oldGenerator := cfg.Local.Settings.MapGenerator
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed reading working directory: %v", err)
	}
	t.Cleanup(func() {
		cfg.Global = oldGlobal
		cfg.Local.Settings.MapGenerator = oldGenerator
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed restoring working directory: %v", err)
		}
	})

	localRoot := t.TempDir()
	sharedRoot := t.TempDir()
	if err := os.Chdir(localRoot); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	cfg.Global.Paths.Folders.ServersRoot = sharedRoot
	cfg.Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	sharedDir := filepath.Join(sharedRoot, constants.DefaultMapGeneratorsDir, "spiral")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("failed creating shared generator dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, constants.MapGenSettingsName), []byte(`{"seed":123}`), 0644); err != nil {
		t.Fatalf("failed writing shared map-gen settings: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, constants.MapSettingsName), []byte(`{"pollution":true}`), 0644); err != nil {
		t.Fatalf("failed writing shared map settings: %v", err)
	}

	displayName, err := applyMapGeneratorSelection("spiral")
	if err != nil {
		t.Fatalf("applyMapGeneratorSelection returned error: %v", err)
	}
	if displayName != "spiral" {
		t.Fatalf("unexpected display name: %q", displayName)
	}
	if cfg.Local.Settings.MapGenerator != "spiral" {
		t.Fatalf("MapGenerator = %q, want spiral", cfg.Local.Settings.MapGenerator)
	}

	genPath, setPath := cfg.GetCachedMapGeneratorFiles("spiral")
	if _, err := os.Stat(genPath); err != nil {
		t.Fatalf("expected cached map-gen settings: %v", err)
	}
	if _, err := os.Stat(setPath); err != nil {
		t.Fatalf("expected cached map settings: %v", err)
	}
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
