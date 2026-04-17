package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ChatWire/cfg"
)

func TestParseRuntimeSelfTestCasesDefault(t *testing.T) {
	got := parseRuntimeSelfTestCases("default")
	if !reflect.DeepEqual(got, defaultRuntimeSelfTestCases) {
		t.Fatalf("unexpected default cases: %#v", got)
	}
}

func TestParseRuntimeSelfTestCasesCustom(t *testing.T) {
	got := parseRuntimeSelfTestCases(" start, restart ,change-map ")
	want := []string{"start", "restart", "change-map"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed cases: got=%#v want=%#v", got, want)
	}
}

func TestChooseRuntimeTestSave(t *testing.T) {
	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = ""
	cfg.Local.Callsign = "srv"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Global.Paths.Folders.Saves = "saves"

	savesDir := filepath.Join(root, cfg.Local.Callsign, cfg.Global.Paths.Folders.FactorioDir, cfg.Global.Paths.Folders.Saves)
	if err := os.MkdirAll(savesDir, 0o755); err != nil {
		t.Fatalf("mkdir saves: %v", err)
	}

	savePath := filepath.Join(savesDir, "candidate.zip")
	f, err := os.Create(savePath)
	if err != nil {
		t.Fatalf("create save: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("test-map/level.dat0")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write(make([]byte, 64*1024)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close save: %v", err)
	}

	got, err := chooseRuntimeTestSave()
	if err != nil {
		t.Fatalf("chooseRuntimeTestSave returned error: %v", err)
	}
	if got != "candidate" {
		t.Fatalf("unexpected save choice: %q", got)
	}
}
