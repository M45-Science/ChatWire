package support

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
)

func TestRegressionRotatedBackupMatchesQuitSave(t *testing.T) {
	resetSupportTestState(t)
	oldLocal, oldGlobal := cfg.Local, cfg.Global
	t.Cleanup(func() { cfg.Local, cfg.Global = oldLocal, oldGlobal })
	root := t.TempDir()
	t.Chdir(root)
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = ""
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Global.Paths.Folders.Saves = "saves"
	cfg.Local.Callsign = "srv"
	cfg.Local.LastSaveBackup = constants.MaxSaveBackups
	if err := os.MkdirAll(cfg.GetSavesFolder(), 0755); err != nil {
		t.Fatal(err)
	}
	makeZip := func(size int, content byte) []byte {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		w, err := z.CreateHeader(&zip.FileHeader{Name: "map/level.dat0", Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(bytes.Repeat([]byte{content}, size)); err != nil {
			t.Fatal(err)
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	current := makeZip(65536, 'A')
	old := makeZip(131072, 'B')
	source := filepath.Join(cfg.GetSavesFolder(), "current.zip")
	backup := filepath.Join(cfg.GetSavesFolder(), "bak-1.zip")
	if err := os.WriteFile(source, current, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backup, old, 0644); err != nil {
		t.Fatal(err)
	}
	input := preProcessFactorioOutput(fmt.Sprintf("0 Info MainLoop.cpp:1: Saving map as %s", source))
	if !handleExitSave(input) {
		t.Fatal("exit save line was not handled")
	}
	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, current) {
		t.Errorf("rotated backup retained stale data: got %d bytes, current save is %d bytes", len(got), len(current))
	}
	if valid, _ := fact.CheckSave(cfg.GetSavesFolder(), "bak-1.zip", false); !valid {
		t.Error("ChatWire rejects its own rotated backup as invalid")
	}
}

func TestRegressionLateExitSaveDoesNotOverrideRewindSelection(t *testing.T) {
	resetSupportTestState(t)
	oldLocal, oldGlobal := cfg.Local, cfg.Global
	t.Cleanup(func() { cfg.Local, cfg.Global = oldLocal, oldGlobal })
	fact.SetAutolaunch(false, false)
	fact.SetUpdateInProgress(false)
	fact.SetModOperationInProgress(false)
	root := t.TempDir()
	t.Chdir(root)
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = ""
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Global.Paths.Folders.Saves = "saves"
	cfg.Local.Callsign = "srv"
	cfg.Local.Name = "alpha"
	cfg.Local.LastSaveBackup = 0
	if err := os.MkdirAll(cfg.GetSavesFolder(), 0755); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, err := z.Create("map/level.dat0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(bytes.Repeat([]byte{'A'}, constants.LevelDatMinSize+1024)); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	oldMap := filepath.Join(cfg.GetSavesFolder(), "old-map.zip")
	for i, name := range []string{oldMap, filepath.Join(cfg.GetSavesFolder(), "rewind.zip")} {
		if err := os.WriteFile(name, b.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
		past := time.Now().Add(-time.Duration(i+1) * time.Hour)
		if err := os.Chtimes(name, past, past); err != nil {
			t.Fatal(err)
		}
	}
	selected := make(chan string, 1)
	fact.StartLifecycleManager(fact.LifecycleHooks{
		LaunchFactorio: func(_ uint64, saveName string) error {
			// Model a buffered save-on-exit log being handled after the controller
			// has prepared alpha_new.zip, but before the launcher scans the directory.
			handleExitSave(preProcessFactorioOutput(fmt.Sprintf("0 Info MainLoop.cpp:1: Saving map as %s", oldMap)))
			_, got, _ := selectLaunchSave(saveName)
			_, pending, _ := selectLaunchSave("")
			if pending != got {
				t.Error("retry did not preserve pending save selection")
			}
			// After readiness clears PendingSave, ordinary restart selection must
			// still prefer the installed rewind over the previous world's save.
			_, nextRestart, _ := GetSaveGame(true)
			if nextRestart != got {
				t.Error("restart after successful rewind would load the previous map")
			}
			if found, _, _ := selectLaunchSave("missing.zip"); found {
				t.Error("missing requested save silently fell back to a different map")
			}

			fact.SetAutolaunch(false, false)
			selected <- got
			return fmt.Errorf("test stops before spawning Factorio")
		},
	})
	defer fact.StopLifecycleManager()
	fact.DoChangeMap("rewind")
	select {
	case got := <-selected:
		want := filepath.Join(cfg.GetSavesFolder(), "alpha_new.zip")
		if got != want {
			t.Fatalf("rewind requested alpha_new.zip but launcher selected %s after late quit-save handling", filepath.Base(got))
		}
	case <-time.After(time.Second):
		t.Fatal("map change did not reach launch")
	}
}
