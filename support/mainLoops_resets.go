package support

import (
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

func startResetDurationLoop() {
	/****************************/
	/* Update time till reset   */
	/****************************/
	go func() {
		ticker := time.NewTicker(time.Minute)

		for glob.ServerRunning {
			<-ticker.C
			if glob.SoftModVersion != constants.Unknown &&
				fact.FactIsRunning &&
				fact.FactorioBooted {
				UpdateDuration()
			}
		}
	}()
}

func startMapResetLoop() {
	/****************************/
	/* Check for map resets	    */
	/****************************/
	go func() {
		for glob.ServerRunning {
			// Run reset checks regardless of Factorio state so scheduled resets
			// can still generate a new map even if the server is down.
			fact.CheckMapReset()
			interval := time.Duration(cfg.Local.Options.MapResetCheckIntervalSec) * time.Second
			if fact.HasResetTime() {
				now := time.Now().UTC()
				until := cfg.Local.Options.NextReset.Sub(now)
				if until <= time.Minute {
					interval = time.Second
				}
			}
			if interval <= 0 {
				interval = time.Minute
			}
			time.Sleep(interval)
		}
	}()
}
