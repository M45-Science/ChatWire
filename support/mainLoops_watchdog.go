package support

import (
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

const (
	watchdogIdleInterval        = 15 * time.Second
	watchdogUnresponsiveTimeout = 2 * time.Minute
)

func startGameWatchdog() {
	/***************
	 * Game watchdog
	 ***************/
	go func() {
		ctx := glob.RuntimeContext()
		lastInterval := time.Duration(0)
		for {
			interval := watchdogPollInterval()
			if lastInterval != 0 && interval != lastInterval {
				// The response counter counts probes, so restart it when the probe
				// cadence changes to preserve the time-based failure threshold.
				glob.ResetNoResponseCount()
			}
			lastInterval = interval
			if !waitForContext(ctx, interval) {
				return
			}

			if fact.FactIsRunning && fact.FactorioBooted {
				// Keep taking low-frequency authoritative samples while paused. That
				// lets the clock and pause state recover when game time advances again.
				nores := glob.IncrementNoResponseCount()
				requestFactorioStatus()
				/* Just in case factorio hangs, bogs down or is flooded */
				if nores >= watchdogFailureThreshold(interval) {
					msg := "Factorio unresponsive for over two minutes... rebooting."
					fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, msg)
					glob.RelaunchThrottle = 0
					glob.ResetNoResponseCount()
					_ = fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionRestartFactorio, Reason: msg})
				}
			}
		}
	}()
}

// requestFactorioStatus probes for the SoftMod protocol while retaining the
// native /time fallback used by scenario and vanilla saves. Until hello is
// acknowledged, both requests are needed: a server without SoftMod cannot
// otherwise initialize its displayed game time.
func requestFactorioStatus() {
	if glob.SoftModVersion == constants.Unknown {
		fact.WriteSoftModCommand("hello", nil)
		fact.WriteFact("/time")
		return
	}
	fact.WriteSoftModCommand("status", nil)
}

func watchdogPollInterval() time.Duration {
	if !fact.FactIsRunning || !fact.FactorioBooted || fact.NumPlayersCurrent() == 0 || fact.PausedTicks > constants.PauseThresh {
		return watchdogIdleInterval
	}
	return constants.WatchdogInterval
}

func watchdogFailureThreshold(interval time.Duration) int {
	if interval <= 0 {
		return 1
	}
	return int((watchdogUnresponsiveTimeout + interval - 1) / interval)
}
