package fact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/util"
)

// Snapshot a requested save before queuing shutdown. Autosaves may rotate while
// a request waits for an update or for the current Factorio process to stop.
func stageMapChange(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid save name %q", name)
	}
	file, err := os.CreateTemp(cfg.GetSavesFolder(), ".cw-rewind-*.tmp.zip")
	if err != nil {
		return "", fmt.Errorf("create rewind snapshot: %w", err)
	}
	snapshot := file.Name()
	if err := file.Close(); err != nil {
		os.Remove(snapshot)
		return "", err
	}
	if err := util.CopyFileAtomic(filepath.Join(cfg.GetSavesFolder(), name+".zip"), snapshot, 0644); err != nil {
		os.Remove(snapshot)
		return "", fmt.Errorf("copy selected save: %w", err)
	}
	if good, _ := CheckSave(filepath.Dir(snapshot), filepath.Base(snapshot), false); !good {
		os.Remove(snapshot)
		return "", fmt.Errorf("selected save %q is invalid", name)
	}
	return snapshot, nil
}

func doChangeMapAfterStop(snapshot string) error {
	// Once startup succeeds, normal restarts choose the newest save again.
	// Promote the selected snapshot without changing the original rewind file.
	now := time.Now()
	if err := os.Chtimes(snapshot, now, now); err != nil {
		return fmt.Errorf("prepare selected save: %w", err)
	}
	destination := filepath.Join(cfg.GetSavesFolder(), cfg.Local.Name+"_new.zip")
	if err := os.Rename(snapshot, destination); err != nil {
		return fmt.Errorf("install selected save: %w", err)
	}
	// Explicit save changes always load the selected file, including on servers
	// that previously generated a scenario which has not yet been started.
	cfg.Local.Settings.NewMap = false
	cfg.Local.PendingSave = filepath.Base(destination)
	cfg.WriteLCfg()
	glob.RelaunchThrottle = 0
	cwlog.DoLogGame("Loading save: %s", destination)
	return nil
}
