package fact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
)

func setupSaveTransitionTest(t *testing.T) string {
	t.Helper()
	resetLifecycleTestState(t)
	resetOperationStatusTestState()
	resetRetryAt, resetRetryFor = time.Time{}, time.Time{}
	oldLocal, oldGlobal := cfg.Local, cfg.Global
	t.Cleanup(func() { resetLifecycleTestState(t); cfg.Local, cfg.Global = oldLocal, oldGlobal })
	root := t.TempDir()
	t.Chdir(root)
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = ""
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Global.Paths.Folders.Saves = "saves"
	cfg.Local.Callsign = "srv"
	cfg.Local.Name = "alpha"
	cfg.Local.PendingSave = ""
	cfg.Local.Settings.NewMap = false
	cfg.Local.Settings.Scenario = ""
	cfg.Local.Settings.MapGenerator = "none"
	cfg.Local.Settings.MapPreset = "default"
	cfg.Local.Settings.Seed = 0
	cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
	cfg.Local.Options.NextReset = time.Time{}
	cfg.Local.Options.ResetHour = 0
	cfg.Local.Channel.ChatChannel = ""
	GameMapPath = ""
	FactorioVersion = constants.Unknown
	for _, dir := range []string{cfg.GetSavesFolder(), cfg.GetModsFolder()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func writeTransitionSave(t *testing.T, path string, content byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, err := z.CreateHeader(&zip.FileHeader{Name: "map/level.dat0", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(bytes.Repeat([]byte{content}, constants.LevelDatMinSize+1024)); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestRegressionRewindPreservesSelectedSnapshot(t *testing.T) {
	setupSaveTransitionTest(t)
	source := filepath.Join(cfg.GetSavesFolder(), "_autosave1.zip")
	want := writeTransitionSave(t, source, 'A')
	lm := newTestLifecycleManager(LifecycleHooks{LaunchFactorio: func(uint64, string) error { return nil }})
	lm.phase, lm.booted, lm.currentGeneration = LifecycleRunning, true, 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()
	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()
	t.Cleanup(func() { lifecycleMu.Lock(); lifecycle = nil; lifecycleMu.Unlock() })
	// Model the selected autosave being overwritten by the running server during shutdown.
	lifecycleSendQuit = func() { writeTransitionSave(t, source, 'B') }
	DoChangeMap("_autosave1")
	req, ok := lm.nextRequest()
	if !ok {
		t.Fatal("map change was not accepted")
	}
	lm.execute(req)
	got, err := os.ReadFile(filepath.Join(cfg.GetSavesFolder(), "alpha_new.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("rewind loaded the save overwritten during shutdown instead of the selected snapshot")
	}
}

func TestRegressionMapChangeCompletesOperationAfterStop(t *testing.T) {
	setupSaveTransitionTest(t)
	writeTransitionSave(t, filepath.Join(cfg.GetSavesFolder(), "candidate.zip"), 'A')
	lm := newTestLifecycleManager(LifecycleHooks{LaunchFactorio: func(uint64, string) error { return nil }})
	lm.phase, lm.booted, lm.currentGeneration = LifecycleRunning, true, 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()
	lm.execute(lifecycleRequest{Request: Request{Kind: ActionChangeMap, SaveName: "candidate"}})
	FactorioVersion = "2.0.76"
	lm.handleReadyEvent(lm.currentGeneration)
	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.token != "" {
		t.Fatalf("successful map change left operation active: title=%q description=%q; manager token=%q", operationStatus.title, operationStatus.description, lm.operationToken)
	}
}

func TestRegressionFailedGenerationPreservesMapAndSchedule(t *testing.T) {
	setupSaveTransitionTest(t)
	cfg.Global.Paths.Binaries.FactBinary = "/bin/false"
	oldMap := filepath.Join(cfg.GetSavesFolder(), "gen-current.zip")
	writeTransitionSave(t, oldMap, 'A')
	cfg.Local.Settings.Seed = 12345
	cfg.Local.Options.ResetInterval = cfg.ResetInterval{Days: 1}
	cfg.Local.Options.NextReset = time.Now().Add(-time.Minute).Round(time.Second)
	wantReset := cfg.Local.Options.NextReset
	if _, err := GenNewMap(); err == nil {
		t.Fatal("expected simulated generation failure")
	}
	if _, err := os.Stat(oldMap); err != nil {
		t.Errorf("failed generation deleted the existing map: %v", err)
	}
	if cfg.Local.Settings.Seed != 12345 {
		t.Errorf("failed generation consumed requested seed: %d", cfg.Local.Settings.Seed)
	}
	if !cfg.Local.Options.NextReset.Equal(wantReset) {
		t.Errorf("failed generation advanced reset deadline from %s to %s", wantReset, cfg.Local.Options.NextReset)
	}
}

func TestRegressionStopTimeoutKeepsProcessTracked(t *testing.T) {
	setupSaveTransitionTest(t)
	lifecycleStopGraceTimeout = time.Millisecond
	lifecycleStopInterruptTimeout = time.Millisecond
	lifecycleStopKillTimeout = time.Millisecond
	lifecycleStopPollInterval = time.Millisecond
	lifecycleProcessAlive = func() bool { return true }
	lifecycleInterruptProcess = func() {}
	lifecycleKillProcess = func() {}
	lm := newTestLifecycleManager(LifecycleHooks{LaunchFactorio: func(uint64, string) error { return nil }})
	lm.phase, lm.currentGeneration = LifecycleRunning, 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()
	SetAutolaunch(true, false)
	if err := lm.executeStop("test failed stop"); err == nil {
		t.Fatal("expected stop failure")
	}
	if lm.GetState().Phase == LifecycleStopped {
		t.Error("controller reports stopped while the old process is still alive")
	}
	if lm.shouldAutoStart() {
		t.Error("controller permits a second launch while the old process is still alive")
	}
	if err := lm.executeStart("manual start after failed stop", ""); err == nil {
		t.Error("manual start was allowed while the previous process is still stopping")
	}
}

func TestGenerationCommitsOnlyValidSave(t *testing.T) {
	for _, valid := range []bool{false, true} {
		t.Run(fmt.Sprintf("valid=%t", valid), func(t *testing.T) {
			root := setupSaveTransitionTest(t)
			old := filepath.Join(cfg.GetSavesFolder(), "gen-old.zip")
			writeTransitionSave(t, old, 'A')
			fixture := filepath.Join(root, "fixture.zip")
			if valid {
				writeTransitionSave(t, fixture, 'B')
			} else if err := os.WriteFile(fixture, []byte("incomplete zip"), 0644); err != nil {
				t.Fatal(err)
			}
			script := filepath.Join(root, "fake-factorio")
			if err := os.WriteFile(script, []byte(fmt.Sprintf("#!/bin/sh\ncp '%s' \"$2\"\n", fixture)), 0755); err != nil {
				t.Fatal(err)
			}
			cfg.Global.Paths.Binaries.FactBinary = script
			cfg.Local.Settings.Seed = 12345
			cfg.Local.Options.ResetInterval = cfg.ResetInterval{Days: 1}
			cfg.Local.Options.NextReset = time.Now().Add(-time.Minute).Round(time.Second)
			due := cfg.Local.Options.NextReset
			name, err := GenNewMap()
			if !valid {
				if err == nil {
					t.Fatal("invalid generated file was accepted")
				}
				if _, err := os.Stat(old); err != nil {
					t.Fatal("failed generation removed old map")
				}
				if cfg.Local.Settings.Seed != 12345 || !cfg.Local.Options.NextReset.Equal(due) || cfg.Local.PendingSave != "" {
					t.Fatal("failed generation changed reset settings")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if good, _ := CheckSave(cfg.GetSavesFolder(), name, false); !good {
				t.Fatal("generated map was not installed")
			}
			if cfg.Local.PendingSave != name || cfg.Local.Settings.Seed != 0 || !cfg.Local.Options.NextReset.After(time.Now()) {
				t.Fatal("successful generation did not commit reset settings")
			}
			if _, err := os.Stat(old); !os.IsNotExist(err) {
				t.Fatal("old generated map was not retired")
			}
		})
	}
}

func TestScheduledResetFailureRetainsDeadlineAndBacksOff(t *testing.T) {
	setupSaveTransitionTest(t)
	cfg.Global.Paths.Binaries.FactBinary = "/bin/false"
	cfg.Local.Options.ResetInterval = cfg.ResetInterval{Days: 1}
	cfg.Local.Options.NextReset = time.Now().Add(-time.Minute).Round(time.Second)
	due := cfg.Local.Options.NextReset
	StartLifecycleManager(LifecycleHooks{LaunchFactorio: func(uint64, string) error { return fmt.Errorf("unexpected launch") }})
	defer StopLifecycleManager()
	CheckMapReset()
	if !cfg.Local.Options.NextReset.Equal(due) {
		t.Fatal("failed scheduled reset consumed its deadline")
	}
	if !resetRetryAt.After(time.Now()) {
		t.Fatal("failed reset did not schedule a retry delay")
	}
	before := resetRetryAt
	CheckMapReset()
	if !resetRetryAt.Equal(before) {
		t.Fatal("failed reset retried immediately")
	}
	// A changed schedule should remain independent of an earlier retry delay.
	cfg.Local.Options.NextReset = due.Add(-time.Second)
	CheckMapReset()
	if !resetRetryFor.Equal(cfg.Local.Options.NextReset) {
		t.Fatal("new scheduled reset was blocked by old retry delay")
	}
}

func TestOneTimeResetCommitsBeforeDeferredLaunch(t *testing.T) {
	for _, offset := range []time.Duration{-time.Minute, time.Hour} {
		t.Run(offset.String(), func(t *testing.T) {
			root := setupSaveTransitionTest(t)
			fixture := filepath.Join(root, "fixture.zip")
			writeTransitionSave(t, fixture, 'A')
			script := filepath.Join(root, "fake-factorio")
			if err := os.WriteFile(script, []byte(fmt.Sprintf("#!/bin/sh\ncp '%s' \"$2\"\n", fixture)), 0755); err != nil {
				t.Fatal(err)
			}
			cfg.Global.Paths.Binaries.FactBinary = script
			cfg.Local.Settings.Scenario = "test-scenario"
			due := time.Now().Add(offset).Round(time.Second)
			cfg.Local.Options.NextReset = due
			lm := newTestLifecycleManager(LifecycleHooks{WithinHours: func() bool { return false }})
			lm.execute(lifecycleRequest{Request: Request{Kind: ActionMapReset}})
			if lm.GetState().LastError != "outside allowed play hours" {
				t.Fatalf("expected launch to be deferred, got %q", lm.GetState().LastError)
			}
			if cfg.Local.PendingSave == "" {
				t.Fatal("deferred launch did not retain the generated save")
			}
			data, err := os.ReadFile(constants.CWLocalConfig)
			if err != nil {
				t.Fatal(err)
			}
			persisted := cfg.Local
			persisted.Settings.NewMap, persisted.PendingSave = false, ""
			if err := json.Unmarshal(data, &persisted); err != nil {
				t.Fatal(err)
			}
			if !persisted.Settings.NewMap || persisted.PendingSave != cfg.Local.PendingSave {
				t.Fatal("ChatWire restart would forget the pending scenario reset")
			}
			if offset < 0 && HasResetTime() {
				t.Fatal("completed reset retained its deadline and would generate another map")
			}
			if offset > 0 && !cfg.Local.Options.NextReset.Equal(due) {
				t.Fatal("manual reset consumed a future scheduled reset")
			}
			lm.hooks.WithinHours = func() bool { return true }
			lm.hooks.LaunchFactorio = func(uint64, string) error { return nil }
			if err := lm.executeStart("deferred scenario reset", ""); err != nil {
				t.Fatal(err)
			}
			FactorioVersion = "2.0.76"
			lm.handleReadyEvent(lm.currentGeneration)
			if cfg.Local.Settings.NewMap || cfg.Local.PendingSave != "" {
				t.Fatal("successful launch left a scenario reset pending")
			}
		})
	}
}

func TestPendingRewindSnapshotsAreIndependentAndCleanedUp(t *testing.T) {
	setupSaveTransitionTest(t)
	source := filepath.Join(cfg.GetSavesFolder(), "candidate.zip")
	first := writeTransitionSave(t, source, 'A')
	lm := newTestLifecycleManager(LifecycleHooks{})
	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()
	t.Cleanup(func() { lifecycleMu.Lock(); lifecycle = nil; lifecycleMu.Unlock() })
	if err := SubmitLifecycleRequest(Request{Kind: ActionChangeMap, SaveName: "candidate"}); err != nil {
		t.Fatal(err)
	}
	second := writeTransitionSave(t, source, 'B')
	if err := SubmitLifecycleRequest(Request{Kind: ActionChangeMap, SaveName: "candidate"}); err != nil {
		t.Fatal(err)
	}
	paths := []string{lm.queue[0].saveSnapshot, lm.queue[1].saveSnapshot}
	for i, want := range [][]byte{first, second} {
		got, err := os.ReadFile(paths[i])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("request %d did not retain its own snapshot", i)
		}
	}
	lm.cancelPendingRequests()
	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("canceled request leaked %s", path)
		}
	}
}

func TestMapChangeFailureAfterStopFinishesOperation(t *testing.T) {
	setupSaveTransitionTest(t)
	writeTransitionSave(t, filepath.Join(cfg.GetSavesFolder(), "candidate.zip"), 'A')
	if err := os.Mkdir(filepath.Join(cfg.GetSavesFolder(), "alpha_new.zip"), 0755); err != nil {
		t.Fatal(err)
	}
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase, lm.booted, lm.currentGeneration = LifecycleRunning, true, 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()
	lm.execute(lifecycleRequest{Request: Request{Kind: ActionChangeMap, SaveName: "candidate"}})
	if lm.GetState().LastError == "" {
		t.Fatal("failed replacement was not reported")
	}
	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.token != "" {
		t.Fatal("failed map change left its operation active")
	}
}

func TestRegressionOldStdoutCloseDoesNotRestartNewGeneration(t *testing.T) {
	setupSaveTransitionTest(t)
	lm := newTestLifecycleManager(LifecycleHooks{LaunchFactorio: func(uint64, string) error { return nil }})
	lm.phase, lm.booted, lm.currentGeneration = LifecycleRunning, true, 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()
	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()
	t.Cleanup(func() { lifecycleMu.Lock(); lifecycle = nil; lifecycleMu.Unlock() })
	// The scanner retains its producing generation even if its EOF callback runs
	// after the lifecycle controller has already launched a replacement process.
	oldGeneration := lm.currentGeneration
	oldScannerClosed := func() { NotifyFactorioHealth(oldGeneration, "stdout-closed", nil) }
	lm.execute(lifecycleRequest{Request: Request{Kind: ActionRestartFactorio}})
	lifecycleProcessAlive = func() bool { return true }
	oldScannerClosed()
	lm.drainAsyncEvents()
	if lm.healthRestartQueued {
		t.Fatalf("old process stdout closure queued an unnecessary restart for new generation %d", lm.currentGeneration)
	}
}
