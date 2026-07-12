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

func TestGetCachedMapGeneratorFilesUsesLocalCacheFolder(t *testing.T) {
	oldGlobal := Global
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed reading working directory: %v", err)
	}
	t.Cleanup(func() {
		Global = oldGlobal
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed restoring working directory: %v", err)
		}
	})

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	genPath, setPath := GetCachedMapGeneratorFiles("spiral")
	wantDir := filepath.Join(tmp, constants.DefaultMapGeneratorsDir, constants.MapGeneratorCacheDir, "spiral")
	if genPath != filepath.Join(wantDir, constants.MapGenSettingsName) {
		t.Fatalf("unexpected cached map-gen path: %q", genPath)
	}
	if setPath != filepath.Join(wantDir, constants.MapSettingsName) {
		t.Fatalf("unexpected cached map-settings path: %q", setPath)
	}
}

func TestCacheMapGeneratorCopiesSharedFilesLocally(t *testing.T) {
	oldGlobal := Global
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed reading working directory: %v", err)
	}
	t.Cleanup(func() {
		Global = oldGlobal
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed restoring working directory: %v", err)
		}
	})

	localRoot := t.TempDir()
	sharedRoot := t.TempDir()
	if err := os.Chdir(localRoot); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	Global.Paths.Folders.ServersRoot = sharedRoot
	Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

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

	genPath, setPath, err := CacheMapGenerator("spiral")
	if err != nil {
		t.Fatalf("CacheMapGenerator returned error: %v", err)
	}
	wantDir := filepath.Join(localRoot, constants.DefaultMapGeneratorsDir, constants.MapGeneratorCacheDir, "spiral")
	if genPath != filepath.Join(wantDir, constants.MapGenSettingsName) {
		t.Fatalf("unexpected cached map-gen path: %q", genPath)
	}
	if setPath != filepath.Join(wantDir, constants.MapSettingsName) {
		t.Fatalf("unexpected cached map-settings path: %q", setPath)
	}

	genData, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("failed reading cached map-gen settings: %v", err)
	}
	if string(genData) != `{"seed":123}` {
		t.Fatalf("unexpected cached map-gen data: %s", genData)
	}
	setData, err := os.ReadFile(setPath)
	if err != nil {
		t.Fatalf("failed reading cached map settings: %v", err)
	}
	if string(setData) != `{"pollution":true}` {
		t.Fatalf("unexpected cached map settings data: %s", setData)
	}
}
