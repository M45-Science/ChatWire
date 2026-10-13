package user

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ChatWire/cfg"
	"ChatWire/modupdate"
)

func setupBuiltinModList(t *testing.T) string {
	t.Helper()
	oldGlobal, oldLocal := cfg.Global, cfg.Local
	t.Cleanup(func() { cfg.Global, cfg.Local = oldGlobal, oldLocal })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir() + "/"
	cfg.Global.Paths.ChatWirePrefix = "cw-"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Local.Callsign = "test"
	root := cfg.GetFactorioFolder()
	for _, name := range []string{"base", "quality", "elevated-rails", "space-age", "core"} {
		directory := filepath.Join(root, "data", name)
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]string{"name": name, "version": "2.0.77"})
		if err := os.WriteFile(filepath.Join(directory, "info.json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestGameModListIncludesBuiltinEnableSettings(t *testing.T) {
	setupBuiltinModList(t)
	if !modupdate.WriteModsList(modupdate.ModListData{Mods: []modupdate.ModData{
		{Name: "base", Enabled: true}, {Name: "quality", Enabled: false},
		{Name: "elevated-rails", Enabled: true}, {Name: "space-age", Enabled: false},
	}}) {
		t.Fatal("write list")
	}
	got, err := gameModListText()
	if err != nil {
		t.Fatal(err)
	}
	want := "Enabled:\nbase (2.0.77) (built-in)\nelevated-rails (2.0.77) (built-in)\n\nDisabled:\nquality (2.0.77) (built-in)\nspace-age (2.0.77) (built-in)"
	if got != want {
		t.Fatalf("list = %q, want %q", got, want)
	}
}

func TestGameModListFindsBuiltinsAfterClear(t *testing.T) {
	setupBuiltinModList(t)
	if !modupdate.WriteModsList(modupdate.ModListData{}) {
		t.Fatal("write empty list")
	}
	before, err := os.ReadFile(filepath.Join(cfg.GetModsFolder(), "mod-list.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := gameModListText()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Built-in mods: not configured:") {
		t.Fatalf("list = %q", got)
	}
	for _, name := range []string{"base", "quality", "elevated-rails", "space-age"} {
		if !strings.Contains(got, name+" (2.0.77) (built-in)") {
			t.Fatalf("missing %s in %q", name, got)
		}
	}
	for _, name := range []string{"core", "recycler"} {
		if strings.Contains(got, name) {
			t.Fatalf("unexpected %s in %q", name, got)
		}
	}
	after, err := os.ReadFile(filepath.Join(cfg.GetModsFolder(), "mod-list.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("listing changed mod settings")
	}
}

func TestGameModListFindsBuiltinsWithoutModsDirectory(t *testing.T) {
	setupBuiltinModList(t)
	got, err := gameModListText()
	if err != nil || !strings.Contains(got, "base (2.0.77) (built-in)") {
		t.Fatalf("list = %q, error = %v", got, err)
	}
}
