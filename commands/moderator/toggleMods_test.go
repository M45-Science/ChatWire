package moderator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/modupdate"
	"github.com/bwmarrin/discordgo"
)

func setupBuiltinToggle(t *testing.T) *discordgo.InteractionCreate {
	t.Helper()
	setupAddMods(t)
	oldRunning, oldBooted := fact.FactIsRunning, fact.FactorioBooted
	t.Cleanup(func() { fact.FactIsRunning, fact.FactorioBooted = oldRunning, oldBooted })
	fact.FactIsRunning, fact.FactorioBooted = false, false
	for _, name := range []string{"base", "space-age", "quality", "elevated-rails"} {
		dir := filepath.Join(cfg.GetFactorioFolder(), "data", name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]string{"name": name, "version": "2.0.77"})
		if err := os.WriteFile(filepath.Join(dir, "info.json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Member: &discordgo.Member{User: &discordgo.User{Username: "tester"}}}}
}

func TestToggleBuiltinAfterClearPersistsOnlyRequestedSetting(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(enableStr(enabled, false), func(t *testing.T) {
			interaction := setupBuiltinToggle(t)
			if !modupdate.WriteModsList(modupdate.ModListData{}) {
				t.Fatal("write empty list")
			}
			result := toggleMod(interaction, "space-age", enabled)
			if !strings.Contains(result, "is now "+enableStr(enabled, true)) {
				t.Fatalf("result = %q", result)
			}
			list, err := modupdate.GetModList()
			if err != nil || len(list.Mods) != 1 || list.Mods[0].Name != "space-age" || list.Mods[0].Enabled != enabled {
				t.Fatalf("list = %+v, error = %v", list, err)
			}
			result = toggleMod(interaction, "space-age", !enabled)
			if !strings.Contains(result, "is now "+enableStr(!enabled, true)) {
				t.Fatalf("result = %q", result)
			}
			list, err = modupdate.GetModList()
			if err != nil || len(list.Mods) != 1 || list.Mods[0].Enabled == enabled {
				t.Fatalf("list = %+v, error = %v", list, err)
			}
		})
	}
}

func TestToggleBuiltinRefusesWhileFactorioRunning(t *testing.T) {
	interaction := setupBuiltinToggle(t)
	fact.FactIsRunning = true
	result := toggleMod(interaction, "space-age", false)
	if !strings.Contains(result, "stop Factorio first") {
		t.Fatalf("result = %q", result)
	}
	if _, err := modupdate.GetModList(); !os.IsNotExist(err) {
		t.Fatalf("unexpected list write: %v", err)
	}
}

func TestToggleBuiltinDoesNotClaimSuccessOnWriteFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires ordinary filesystem permissions")
	}
	interaction := setupBuiltinToggle(t)
	if !modupdate.WriteModsList(modupdate.ModListData{Mods: []modupdate.ModData{{Name: "space-age", Enabled: true}}}) {
		t.Fatal("write list")
	}
	mods := cfg.GetModsFolder()
	if err := os.Chmod(mods, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(mods, 0755) })
	before := len(modupdate.ModHistory.History)
	result := toggleMod(interaction, "space-age", false)
	if !strings.Contains(result, "Unable to save mod list") || strings.Contains(result, "is now disabled") {
		t.Fatalf("result = %q", result)
	}
	list, err := modupdate.GetModList()
	if err != nil || len(list.Mods) != 1 || !list.Mods[0].Enabled {
		t.Fatalf("list = %+v, error = %v", list, err)
	}
	if len(modupdate.ModHistory.History) != before {
		t.Fatal("recorded a change that was not saved")
	}
}

func TestToggleBaseCannotDisableRequiredMod(t *testing.T) {
	interaction := setupBuiltinToggle(t)
	result := toggleMod(interaction, "base", false)
	if !strings.Contains(result, "cannot be toggled") {
		t.Fatalf("result = %q", result)
	}
	if _, err := modupdate.GetModList(); !os.IsNotExist(err) {
		t.Fatalf("unexpected list write: %v", err)
	}
}
