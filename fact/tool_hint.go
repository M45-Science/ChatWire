package fact

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/glob"
	"ChatWire/util"
)

// ClaimToolHint atomically claims a reminder across all server instances.
// Keep this timestamp monotonic when stale player snapshots are later saved.
func ClaimToolHint(name string, now time.Time) (bool, error) {
	if name == "" || len(name) > constants.MaxNameLength {
		return false, nil
	}
	name = strings.ToLower(name)
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()
	path := filepath.Join(cfg.Global.Paths.Folders.ServersRoot, cfg.Global.Paths.DataFiles.DBFile)
	lock, err := util.AcquireFileLock(path+".lock", cfg.Local.Callsign, playerDBLockTimeout)
	if err != nil {
		return false, err
	}
	defer lock.Release()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	players, _, err := decodePlayerDatabase(data)
	if err != nil {
		return false, err
	}
	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()
	local := glob.PlayerList[name]
	player := players[name]
	if player == nil {
		if local == nil {
			return false, nil
		}
		copy := *local
		player = &copy
		players[name] = player
	}
	if local != nil && local.LastToolHint > player.LastToolHint {
		player.LastToolHint = local.LastToolHint
	}
	level := player.Level
	// A stale lower-rank snapshot must not shorten a higher-rank cooldown.
	if local != nil && local.Level > level {
		level = local.Level
	}
	if player.Level < 0 || (local != nil && local.Level < 0) {
		return false, nil
	}
	interval := 24 * time.Hour
	if level > 1 {
		interval = 30 * 24 * time.Hour
	}
	if player.LastToolHint != 0 && now.Unix()-player.LastToolHint < int64(interval/time.Second) {
		return false, nil
	}
	player.LastToolHint = now.Unix()
	encoded, err := encodePlayerDatabase(players, cfg.Global.Paths.DataFiles.DBFormat)
	if err != nil {
		return false, err
	}
	if err := util.WriteBytesAtomic(path, encoded, 0644); err != nil {
		return false, err
	}
	if local != nil {
		local.LastToolHint = player.LastToolHint
	}
	return true, nil
}
