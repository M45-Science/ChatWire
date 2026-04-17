package support

import (
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

func startGameWatchdog() {
	/***************
	 * Game watchdog
	 ***************/
	go func() {
		time.Sleep(time.Second * 1)
		for glob.ServerRunning {

			time.Sleep(constants.WatchdogInterval)

			if fact.FactIsRunning && fact.FactorioBooted {

				/* If the game isn't paused, check game time */
				nores := 0
				if fact.PausedTicks <= constants.PauseThresh {

					glob.NoResponseCount = glob.NoResponseCount + 1
					nores = glob.NoResponseCount

					fact.WriteFact("/time")
				}
				/* Just in case factorio hangs, bogs down or is flooded */
				if nores == 120 {
					msg := "Factorio unresponsive for over two minutes... rebooting."
					fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, msg)
					glob.RelaunchThrottle = 0
					_ = fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionRestartFactorio, Reason: msg})
				}
			}
		}
	}()
}
