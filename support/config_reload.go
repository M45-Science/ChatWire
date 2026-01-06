package support

import (
	"sync"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/util"
)

var configReloadLock sync.Mutex

func ReloadConfigFiles(source string) {
	configReloadLock.Lock()
	defer configReloadLock.Unlock()

	cwlog.DoLogCW("Reloading config files (%s)...", source)

	if !cfg.ReadGCfg() {
		cwlog.DoLogCW("Reload config failed: unable to read global config.")
		return
	}
	if !cfg.ReadLCfg() {
		cwlog.DoLogCW("Reload config failed: unable to read local config.")
		return
	}

	cfg.WriteGCfg()
	cfg.WriteLCfg()
	util.SetTempFilePrefix(cfg.Local.Callsign + "-")

	ConfigSoftMod()
	fact.GenerateFactorioConfig()
	fact.DoUpdateChannelName()

	cwlog.DoLogCW("Config files reloaded.")
}
