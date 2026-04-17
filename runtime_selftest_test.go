package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/fact"
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

func TestWaitForLifecycleRestartRequiresCycle(t *testing.T) {
	states := []fact.State{
		{Phase: fact.LifecycleRunning, Booted: true, PID: 100, Since: time.Unix(100, 0)},
		{Phase: fact.LifecycleStopping, Booted: false, PID: 100, Since: time.Unix(101, 0)},
		{Phase: fact.LifecycleRunning, Booted: true, PID: 200, Since: time.Unix(102, 0)},
	}
	idx := 0
	getState := func() fact.State {
		if idx >= len(states) {
			return states[len(states)-1]
		}
		s := states[idx]
		idx++
		return s
	}

	before := states[0]
	if err := waitForLifecycleRestartWithGetter(2*time.Second, before, getState); err != nil {
		t.Fatalf("waitForLifecycleRestartWithGetter returned error: %v", err)
	}
}

func TestWaitForLifecycleRestartTimesOutWithoutCycle(t *testing.T) {
	state := fact.State{Phase: fact.LifecycleRunning, Booted: true, PID: 100, Since: time.Unix(100, 0)}
	getState := func() fact.State { return state }

	err := waitForLifecycleRestartWithGetter(50*time.Millisecond, state, getState)
	if err == nil {
		t.Fatal("expected timeout when restart cycle never occurs")
	}
}
