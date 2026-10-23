package controlruntime

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ChatWire/cfg"
	"ChatWire/modupdate"
	"ChatWire/webcontrol"
)

func TestWebSpaceAgeModsAreDiscoveredAndEditedSafely(t *testing.T) {
	oldGlobal, oldLocal := cfg.Global, cfg.Local
	t.Cleanup(func() { cfg.Global, cfg.Local = oldGlobal, oldLocal })
	root := t.TempDir()
	t.Chdir(root)
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = "cw-"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Local.Callsign = "a"
	deps := map[string][]string{"base": {}, "elevated-rails": {"base >= 2.1.0"}, "quality": {"base >= 2.1.0", "recycler >= 2.1.0"}, "recycler": {"base >= 2.1.0"}, "space-age": {"base >= 2.1.0", "elevated-rails >= 2.1.0", "+ quality >= 2.1.0", "recycler >= 2.1.0"}}
	for name, dep := range deps {
		dir := filepath.Join(cfg.GetFactorioFolder(), "data", name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]any{"name": name, "version": "2.1.23", "dependencies": dep})
		if err := os.WriteFile(filepath.Join(dir, "info.json"), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Start without mod-list.json, as after installation/clearing configuration.
	rt, err := New(webcontrol.InstanceConfig{ID: "a", Socket: filepath.Join(root, "a.sock"), StateDir: filepath.Join(root, "state")})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/internal/v1/mods", nil)
	req.Header.Set("X-ChatWire-Instance", "a")
	req.Header.Set("X-ChatWire-Actor", `{"id":"test"}`)
	recorder := httptest.NewRecorder()
	rt.Handler().ServeHTTP(recorder, req)
	if recorder.Code != 200 {
		t.Fatalf("inventory failed: %s", recorder.Body.String())
	}
	var inventory struct {
		Mods []modupdate.ModInventoryEntry `json:"mods"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Mods) != 5 {
		t.Fatalf("missing bundled components: %s", recorder.Body.String())
	}
	if _, err := editMods(Params{Operation: "enable", Mods: []string{"space-age", "quality"}}); err != nil {
		t.Fatal(err)
	}
	list, err := modupdate.GetModList()
	if err != nil {
		t.Fatal(err)
	}
	enabled := map[string]bool{}
	for _, mod := range list.Mods {
		enabled[mod.Name] = mod.Enabled
	}
	for name := range deps {
		if !enabled[name] {
			t.Fatalf("%s not enabled: %v", name, list)
		}
	}
	filename := filepath.Join(cfg.GetModsFolder(), "mod-list.json")
	before, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, params := range []Params{{Operation: "disable", Mods: []string{"recycler"}}, {Operation: "disable", Mods: []string{"base"}}, {Operation: "remove", Mods: []string{"space-age"}}, {Operation: "version", Mods: []string{"recycler"}, Version: "2.1.22"}} {
		if _, err := editMods(params); err == nil {
			t.Fatalf("accepted unsafe edit: %+v", params)
		}
		after, _ := os.ReadFile(filename)
		if !bytes.Equal(before, after) {
			t.Fatalf("failed edit changed mod list: %+v", params)
		}
	}
	// Quality is recommended rather than required by Space Age 2.1.
	if _, err := editMods(Params{Operation: "disable", Mods: []string{"quality"}}); err != nil {
		t.Fatal(err)
	}
}
