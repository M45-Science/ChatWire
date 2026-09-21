package support

import (
	"ChatWire/glob"
	"errors"
	"sync"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/util"
)

var configReloadLock sync.Mutex

func ReloadConfigFiles(source string) {
	glob.ControlLock.Lock()
	defer glob.ControlLock.Unlock()
	_ = ReloadConfigFilesResult(source)
}

// ReloadConfigFilesResult is used by adapters already holding ControlLock.
func ReloadConfigFilesResult(source string) error {
	configReloadLock.Lock()
	defer configReloadLock.Unlock()

	cwlog.DoLogCW("Reloading config files (%s)...", source)

	if !cfg.ReadGCfg() {
		cwlog.DoLogCW("Reload config failed: unable to read global config.")
		return errors.New("unable to reload configuration")
	}
	if !cfg.ReadLCfg() {
		cwlog.DoLogCW("Reload config failed: unable to read local config.")
		return errors.New("unable to reload configuration")
	}

	cfg.WriteGCfg()
	cfg.WriteLCfg()
	util.SetTempFilePrefix(cfg.Local.Callsign + "-")

	ConfigSoftMod()
	if !fact.GenerateFactorioConfig() {
		return errors.New("configuration loaded but Factorio settings could not be applied")
	}
	fact.DoUpdateChannelName()

	cwlog.DoLogCW("Config files reloaded.")
	return nil
}
