package modupdate

import (
	"ChatWire/cfg"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func builtinFixture(t *testing.T, version string) {
	t.Helper()
	oldGlobal, oldLocal := cfg.Global, cfg.Local
	t.Cleanup(func() { cfg.Global, cfg.Local = oldGlobal, oldLocal })
	root := t.TempDir()
	t.Chdir(root)
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = "cw-"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Local.Callsign = "a"
	deps := map[string][]string{"base": {}, "elevated-rails": {"base >= 2.0.0"}, "quality": {"base >= 2.0.0"}, "space-age": {"base >= 2.0.0", "elevated-rails >= 2.0.0", "quality >= 2.0.0"}}
	if version == "2.1.23" {
		deps["recycler"] = []string{"base >= 2.1.0"}
		deps["quality"] = []string{"base >= 2.1.0", "recycler >= 2.1.0"}
		deps["space-age"] = []string{"base >= 2.1.0", "elevated-rails >= 2.1.0", "+ quality >= 2.1.0", "recycler >= 2.1.0"}
	}
	for name, dep := range deps {
		dir := filepath.Join(cfg.GetFactorioFolder(), "data", name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(modZipInfo{Name: name, Title: name, Version: version, Dependencies: dep})
		if err := os.WriteFile(filepath.Join(dir, "info.json"), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSpaceAgeBuiltinDependenciesAcrossReleases(t *testing.T) {
	for _, version := range []string{"2.0.77", "2.1.23"} {
		t.Run(version, func(t *testing.T) {
			builtinFixture(t, version)
			list, err := SetBuiltinModEnabled(ModListData{}, "space-age", true)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateBuiltinModDependencies(list); err != nil {
				t.Fatal(err)
			}
			enabled := map[string]bool{}
			for _, m := range list.Mods {
				enabled[m.Name] = m.Enabled
			}
			for _, name := range []string{"base", "elevated-rails", "quality", "space-age"} {
				if !enabled[name] {
					t.Errorf("%s not enabled", name)
				}
			}
			if enabled["recycler"] != (version == "2.1.23") {
				t.Fatalf("wrong Recycler state: %v", enabled)
			}
			if _, err := SetBuiltinModEnabled(list, "base", false); err == nil {
				t.Fatal("disabled base")
			}
			bad, err := SetBuiltinModEnabled(list, "elevated-rails", false)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateBuiltinModDependencies(bad); err == nil {
				t.Fatal("accepted missing Space Age dependency")
			}
		})
	}
}

func TestSpaceAgeRecommendationCanStayDisabled(t *testing.T) {
	builtinFixture(t, "2.1.23")
	list, err := SetBuiltinModEnabled(ModListData{Mods: []ModData{{Name: "quality", Enabled: false}}}, "space-age", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range list.Mods {
		if m.Name == "quality" && m.Enabled {
			t.Fatal("overrode disabled recommendation")
		}
	}
	if err := ValidateBuiltinModDependencies(list); err != nil {
		t.Fatal(err)
	}
}

func TestBuiltinInventoryIncludesUnconfiguredRecycler(t *testing.T) {
	builtinFixture(t, "2.1.23")
	if !WriteModsList(ModListData{Mods: []ModData{{Name: "base", Enabled: true}, {Name: "space-age", Enabled: false}}}) {
		t.Fatal("write list")
	}
	entries, err := GetModInventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("inventory = %v", entries)
	}
	found := false
	for _, m := range entries {
		if m.Name == "recycler" {
			found = true
			if !m.Builtin || m.Configured || m.Version != "2.1.23" {
				t.Fatalf("Recycler = %+v", m)
			}
		}
	}
	if !found {
		t.Fatal("missing Recycler")
	}
}
