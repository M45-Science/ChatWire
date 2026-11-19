package support

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/util"
)

func warningMutePath() string {
	return filepath.Join(cfg.Global.Paths.Folders.ServersRoot, "warning-ping-mutes.json")
}

func readWarningMutes() (map[string]int64, error) {
	m := make(map[string]int64)
	data, err := os.ReadFile(warningMutePath())
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]int64)
	}
	return m, nil
}

// SetWarningPingMute applies to all servers sharing the servers root and survives restarts.
func SetWarningPingMute(player string, minutes int) (time.Time, error) {
	player = strings.ToLower(strings.TrimSpace(player))
	if player == "" || (minutes < 1 || minutes > 153722867) {
		return time.Time{}, fmt.Errorf("provide a player name and a positive number of minutes")
	}
	path := warningMutePath()
	lock, err := util.AcquireFileLock(path+".lock", cfg.Local.Callsign, 5*time.Second)
	if err != nil {
		return time.Time{}, err
	}
	defer lock.Release()
	m, err := readWarningMutes()
	if err != nil {
		return time.Time{}, err
	}
	now := time.Now().UTC()
	for name, until := range m {
		if until <= now.Unix() {
			delete(m, name)
		}
	}
	until := now.Add(time.Duration(minutes) * time.Minute)
	m[player] = until.Unix()
	data, err := json.Marshal(m)
	if err != nil {
		return time.Time{}, err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".warning-ping-mutes-*")
	if err != nil {
		return time.Time{}, err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return time.Time{}, err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return time.Time{}, err
	}
	return until, nil
}

func warningPingMuted(player string, now time.Time) bool {
	if strings.TrimSpace(player) == "" {
		return false
	}
	m, err := readWarningMutes()
	if err != nil {
		cwlog.DoLogCW("Warning ping mute lookup: %v", err)
		return false
	}
	return m[strings.ToLower(strings.TrimSpace(player))] > now.Unix()
}
