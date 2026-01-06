package support

import (
	"math/rand/v2"
	"os"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func startModUpdateLoop() {
	/****************************/
	/* Check for mod update     */
	/****************************/
	//Every 3 hours
	go func() {

		if *glob.ProxyURL != "" {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()

			for glob.ServerRunning {
				<-ticker.C
				if !cfg.Local.Options.ModUpdate {
					continue
				}

				cTime := time.Now()
				if cTime.Minute() != 0 {
					continue
				}
				if cTime.Hour()%3 == 0 {
					glob.UpdatersLock.Lock()
					modupdate.CheckMods(false, false)
					glob.UpdatersLock.Unlock()
				}
			}
		} else {

			for glob.ServerRunning {
				time.Sleep(time.Hour * 3)

				if cfg.Local.Options.ModUpdate {
					glob.UpdatersLock.Lock()
					modupdate.CheckMods(false, false)
					glob.UpdatersLock.Unlock()
				}

				//Add 0 to 5 minutes of sleep
				time.Sleep(time.Microsecond * time.Duration((rand.Float64() * 60000000 * 5.0)))
			}
		}
	}()
}

func startModPackCleanupLoop() {
	/****************************/
	/* Auto delete modpack files
	 * at the set expire time
	/****************************/
	go func() {
		for glob.ServerRunning {

			time.Sleep(time.Minute)
			numItems := len(cfg.Local.ModPackList)

			if numItems > 0 {
				changed := false
				kept := make([]cfg.ModPackData, 0, numItems)

				for _, item := range cfg.Local.ModPackList {
					expired := item.Path == "" || time.Since(item.Created) > (constants.ModPackLifeMins*time.Minute)
					if !expired {
						kept = append(kept, item)
						continue
					}
					changed = true
					if item.Path != "" {
						if err := os.Remove(item.Path); err != nil {
							cwlog.DoLogCW("Unable to delete expired modpack!")
						} else {
							cwlog.DoLogCW("Deleted expired modpack: " + item.Path)
						}
					}
				}

				if changed {
					cfg.Local.ModPackList = kept
					cfg.WriteLCfg()
				}
			}
		}
	}()
}
