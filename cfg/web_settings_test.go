package cfg

import (
	"ChatWire/constants"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func webFixture(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "server"), 0700); e != nil {
		t.Fatal(e)
	}
	t.Chdir(filepath.Join(root, "server"))
	if e := os.WriteFile(constants.CWLocalConfig, []byte(`{"Name":"Alpha","Callsign":"a","Port":34197,"Settings":{"AFKMin":15},"Options":{"Speed":1}}`), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(constants.CWGlobalConfig, []byte(`{"GroupName":"M45","Options":{"RconOffset":1000},"Discord":{"Token":"super-secret"}}`), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestWebSettingsRevisionRedactionAndAtomicValidation(t *testing.T) {
	webFixture(t)
	g, e := WebRead(true)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(g)
	if string(b) == "" {
		t.Fatal("missing settings")
	}
	if g.Values["Discord.Token"].(map[string]bool)["configured"] != true {
		t.Fatal("missing secret status")
	}
	v, _ := WebRead(false)
	_, e = WebPatch(false, false, v.Revision, map[string]json.RawMessage{"Name": json.RawMessage(`"Beta"`), "Port": json.RawMessage(`999999`)})
	if e == nil {
		t.Fatal("invalid patch accepted")
	}
	same, _ := WebRead(false)
	if same.Revision != v.Revision {
		t.Fatal("partial mutation")
	}
	updated, e := WebPatch(false, false, v.Revision, map[string]json.RawMessage{"Name": json.RawMessage(`"Beta"`)})
	if e != nil {
		t.Fatal(e)
	}
	if updated.Values["Name"] != "Beta" {
		t.Fatal(updated)
	}
	if _, e = WebPatch(false, false, v.Revision, map[string]json.RawMessage{"Name": json.RawMessage(`"Gamma"`)}); e != ErrRevision {
		t.Fatal("stale revision accepted")
	}
	if _, e = WebPatch(false, false, updated.Revision, map[string]json.RawMessage{"Callsign": json.RawMessage(`"b"`)}); e == nil {
		t.Fatal("admin-only field accepted")
	}
}
func TestStaleRuntimeWritePreservesWebEdit(t *testing.T) {
	webFixture(t)
	runtime := local{Name: "Alpha", Callsign: "a", Port: 34197}
	rememberConfig(constants.CWLocalConfig, runtime)
	v, _ := WebRead(false)
	if _, e := WebPatch(false, false, v.Revision, map[string]json.RawMessage{"Name": json.RawMessage(`"Web name"`)}); e != nil {
		t.Fatal(e)
	}
	runtime.Port = 34198
	if e := writeConfigMerged(constants.CWLocalConfig, runtime); e != nil {
		t.Fatal(e)
	}
	got, _ := WebRead(false)
	if got.Values["Name"] != "Web name" || got.Values["Port"] != float64(34198) {
		t.Fatalf("lost edit: %#v", got.Values)
	}
}

func TestWebSchemaRequiresExplicitFieldClassification(t *testing.T) {
	for _, global := range []bool{false, true} {
		scope := "local"
		if global {
			scope = "global"
		}
		for _, field := range WebSchema(global) {
			if !classifiedWebFields[scope+"."+field.Key] {
				t.Errorf("classify %s.%s before exposing it", scope, field.Key)
			}
		}
	}
}

func TestIntervalPatchSchedulesAndDisablesNextReset(t *testing.T) {
	webFixture(t)
	v, _ := WebRead(false)
	next, e := WebPatch(false, false, v.Revision, map[string]json.RawMessage{"Options.ResetInterval.Days": json.RawMessage(`7`)})
	if e != nil {
		t.Fatal(e)
	}
	date, e := time.Parse(time.RFC3339, next.Values["Options.NextReset"].(string))
	if e != nil || date.Before(time.Now().Add(6*24*time.Hour)) {
		t.Fatal("next reset was not calculated")
	}
	off, e := WebPatch(false, false, next.Revision, map[string]json.RawMessage{"Options.ResetInterval.Days": json.RawMessage(`0`)})
	if e != nil {
		t.Fatal(e)
	}
	date, e = time.Parse(time.RFC3339, off.Values["Options.NextReset"].(string))
	if e != nil || !date.IsZero() {
		t.Fatal("zero interval did not disable reset")
	}
}
