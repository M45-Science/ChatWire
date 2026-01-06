package support

import (
	"time"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/watcher"
)

func startDBFileWatcher() {
	/************************************
	 * Database file modification watching
	 ************************************/
	go func() {
		filePath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
		reload := newDebounce(time.Second, func() {
			//cwlog.DoLogCW("Database file modified, loading.")
			fact.LoadPlayers(false, false, false)
		})
		watcher.Watch(filePath, 5*time.Second, &glob.ServerRunning, func() {
			time.Sleep(time.Second)
			reload.trigger()
		})
	}()
}
