package support

import (
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/util"
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

func startGlobalConfigWatcher() {
	/****************************************
	 * Global config file modification watching
	 ****************************************/
	go func() {
		reload := newDebounce(time.Second, func() {
			if cfg.ReadGCfg() {
				ConfigSoftMod()
				fact.GenerateFactorioConfig()
				fact.DoUpdateChannelName()
			}
		})
		watcher.Watch(constants.CWGlobalConfig, 5*time.Second, &glob.ServerRunning, func() {
			time.Sleep(time.Second)
			reload.trigger()
		})
	}()
}

func startLocalConfigWatcher() {
	/****************************************
	 * Local config file modification watching
	 ****************************************/
	go func() {
		reload := newDebounce(time.Second, func() {
			if cfg.ReadLCfg() {
				util.SetTempFilePrefix(cfg.Local.Callsign + "-")
				ConfigSoftMod()
				fact.GenerateFactorioConfig()
				fact.DoUpdateChannelName()
			}
		})
		watcher.Watch(constants.CWLocalConfig, 5*time.Second, &glob.ServerRunning, func() {
			time.Sleep(time.Second)
			reload.trigger()
		})
	}()
}
