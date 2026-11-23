package support

import (
	"os"
	"time"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
)

func startDBFileWatcher() {
	go func() {
		ctx := glob.RuntimeContext()
		filePath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
		var loaded os.FileInfo
		for waitForContext(ctx, 2*time.Second) {
			current, err := os.Stat(filePath)
			if err == nil && (loaded == nil || !os.SameFile(current, loaded) ||
				current.Size() != loaded.Size() || !current.ModTime().Equal(loaded.ModTime())) {
				// Keep the stat from before the read so a concurrent replacement
				// during the reload is detected on the next poll.
				if fact.LoadPlayers(false, false, false) {
					loaded = current
				}
			}
			fact.RetryNativeModeration()
			fact.ReconcilePlayerBans()
		}
	}()
}
