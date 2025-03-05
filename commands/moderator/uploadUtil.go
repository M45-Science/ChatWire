package moderator

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/support"
	"strings"
	"sync"
	"time"
)

const (
	saveGameName       = "save-game"
	MaxModSettingsSize = 1024 * 1024 //1MB
	MaxModListSize     = 1024 * 1024 //1MB
)

var (
	UploadLock                           sync.Mutex
	foundOption, foundSave, foundModList bool
	errMsgDelay                          = time.Second * 3
)

func showSyncMods() string {
	buf := ""
	modList, mErr := support.GetGameMods()
	if mErr == nil && modList != nil {
		for _, mod := range modList.Mods {
			if strings.EqualFold(mod.Name, "base") {
				continue
			}
			if strings.EqualFold(mod.Name, "elevated-rails") {
				continue
			}
			if strings.EqualFold(mod.Name, "quality") {
				continue
			}
			if strings.EqualFold(mod.Name, "space-age") {
				continue
			}
			if !mod.Enabled {
				continue
			}
			if buf != "" {
				buf = buf + ", "
			}
			if mod.Enabled {
				buf = buf + strings.TrimSuffix(mod.Name, ".zip")
			}
		}
	}

	if buf == "" {
		buf = strings.Join(support.GetModFiles(), ", ")
	}

	return buf
}

func stopWaitFact(msg string) {
	if fact.FactorioBooted || fact.FactIsRunning {

		/* Turn off skip reset flag regardless of reset reason */
		if cfg.Local.Options.SkipReset {
			cfg.Local.Options.SkipReset = false
			cfg.WriteLCfg()
		}

		cfg.Local.Options.SkipReset = false
		fact.QueueReboot = false      //Skip queued reboot
		fact.QueueFactReboot = false  //Skip queued reboot
		fact.DoUpdateFactorio = false //Skip queued updates
		cfg.WriteLCfg()

		fact.SetAutolaunch(false, false)
		fact.QuitFactorio(msg)
		fact.WaitFactQuit(false)
	}
}
