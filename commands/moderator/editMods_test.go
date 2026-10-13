package moderator

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/modupdate"
)

func setupAddMods(t *testing.T) {
	t.Helper()
	oldGlobal, oldLocal, oldHistory := cfg.Global, cfg.Local, modupdate.ModHistory
	t.Cleanup(func() { cfg.Global, cfg.Local, modupdate.ModHistory = oldGlobal, oldLocal, oldHistory })
	root := t.TempDir()
	t.Chdir(root)
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = "cw-"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Local.Callsign = "test"
	if err := os.MkdirAll(cfg.GetModsFolder(), 0755); err != nil {
		t.Fatal(err)
	}
}

func writeAddedMod(t *testing.T, name string) {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, err := z.Create(name + "_1.0.0/info.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(f).Encode(map[string]string{"name": name, "version": "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.GetModsFolder(), name+"_1.0.0.zip"), b.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestAddModsContinuesAfterMissingAndInstalledEntries(t *testing.T) {
	setupAddMods(t)
	writeAddedMod(t, "existing")
	calls := 0
	resolve := func(name string) (string, bool) {
		if name == "missing" {
			return "Mod not found: missing", false
		}
		return name, true
	}
	got := addMods("first,missing,existing,last,first,", "tester", resolve, func() (bool, error) {
		calls++
		list, err := modupdate.GetModList()
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Mods) != 2 || list.Mods[0].Name != "first" || list.Mods[1].Name != "last" {
			t.Fatalf("list = %+v", list)
		}
		writeAddedMod(t, "first")
		writeAddedMod(t, "last")
		return true, nil
	})
	if calls != 1 {
		t.Fatalf("download calls = %d", calls)
	}
	for _, want := range []string{"Mod not found: missing", "existing: already installed", "first: installed", "last: installed"} {
		if !strings.Contains(got, want) {
			t.Fatalf("response %q missing %q", got, want)
		}
	}
}

func TestAddModsReportsFailureAndRetriesListedMod(t *testing.T) {
	setupAddMods(t)
	resolve := func(name string) (string, bool) { return name, true }
	got := addMods("retry", "tester", resolve, func() (bool, error) { return false, errors.New("download unavailable") })
	if !strings.Contains(got, "download unavailable") || !strings.Contains(got, "retry: not installed") {
		t.Fatalf("response = %q", got)
	}
	got = addMods("retry", "tester", resolve, func() (bool, error) { writeAddedMod(t, "retry"); return true, nil })
	if !strings.Contains(got, "retry: installed") {
		t.Fatalf("retry response = %q", got)
	}
	list, err := modupdate.GetModList()
	if err != nil || len(list.Mods) != 1 {
		t.Fatalf("retry list = %+v, %v", list, err)
	}
}

func TestAddModsDoesNotClaimSuccessWithoutArchive(t *testing.T) {
	setupAddMods(t)
	got := addMods("unresolved", "tester", func(n string) (string, bool) { return n, true }, func() (bool, error) { return true, nil })
	if !strings.Contains(got, "unresolved: not installed") {
		t.Fatalf("response = %q", got)
	}
}

func TestAddModsStopsOnUnreadableOrUnwritableList(t *testing.T) {
	for _, malformed := range []bool{true, false} {
		t.Run(map[bool]string{true: "malformed", false: "unwritable"}[malformed], func(t *testing.T) {
			setupAddMods(t)
			path := filepath.Join(cfg.GetModsFolder(), constants.ModListName)
			if malformed {
				if err := os.WriteFile(path, []byte("invalid json"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			got := addMods("new", "tester", func(n string) (string, bool) {
				if !malformed {
					if err := os.Mkdir(path, 0755); err != nil {
						t.Fatal(err)
					}
				}
				return n, true
			}, func() (bool, error) { t.Fatal("must not download"); return false, nil })
			if !strings.Contains(got, "Unable to") {
				t.Fatalf("response = %q", got)
			}
		})
	}
}

func TestModInputNameProvidedBatch(t *testing.T) {
	// The reported batch includes a percent-encoded space and mixed-case names.
	names := strings.Split("informatron,jetpack,grappling-gun,bullet-trails,shield-projector,aai-vehicles-ironclad,combat-mechanics-overhaul,aai-signal-transmission,aai-containers,aai-loaders,aai-industry,robot_attrition,space-exploration,space-exploration-graphics,space-exploration-graphics-2,space-exploration-graphics-3,space-exploration-graphics-4,space-exploration-graphics-5,space-exploration-postprocess,space-exploration-menu-simulations,alien-biomes,alien-biomes-graphics,EvenDistributionLite,RateCalculator,factoryplanner,dqol-resource-monitor,Milestones,better-victory-screen,Orphan Finder,pickitup,squeak-through-2,Concrete-Tints,color_whole_train,FluidWagonColorMask,textplates,even-more-text-plates-2_0,flib", ",")
	for _, name := range names {
		input := "https://mods.factorio.com/mod/" + strings.ReplaceAll(name, " ", "%20")
		got, err := modInputName(input)
		if err != nil || got != name {
			t.Fatalf("modInputName(%q) = %q, %v; want %q", input, got, err, name)
		}
	}
}

func TestModInputNameURLVariants(t *testing.T) {
	for _, input := range []string{"Orphan Finder", "https://mods.factorio.com/mod/Orphan%20Finder/downloads?from=search#top", "http://mods.factorio.com/download/Orphan%20Finder/123"} {
		got, err := modInputName(input)
		if err != nil || got != "Orphan Finder" {
			t.Fatalf("modInputName(%q) = %q, %v", input, got, err)
		}
	}
	for _, input := range []string{"", "https://example.com/mod/flib", "https://mods.factorio.com/mod/", "https://mods.factorio.com/mod/%zz"} {
		if _, err := modInputName(input); err == nil {
			t.Fatalf("expected invalid input: %q", input)
		}
	}
}

func TestAddModsReportsPartialDownloadAsIncomplete(t *testing.T) {
	setupAddMods(t)
	got := addMods("first,last", "tester", func(n string) (string, bool) { return n, true }, func() (bool, error) {
		writeAddedMod(t, "first")
		return false, errors.New("dependency download failed")
	})
	for _, want := range []string{"first: downloaded, but the batch failed", "last: not installed", "dependency download failed"} {
		if !strings.Contains(got, want) {
			t.Fatalf("response %q missing %q", got, want)
		}
	}
}
