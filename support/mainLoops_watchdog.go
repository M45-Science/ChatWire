package support

import (
	"fmt"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
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

			/* Factorio not running */
			if !fact.FactIsRunning && !fact.DoUpdateFactorio {

				if !fact.FactorioBooted && !fact.FactorioBootedAt.IsZero() {
					if time.Since(fact.FactorioBootedAt) < time.Minute*2 {
						glob.CrashLoopCount++
					} else {
						glob.CrashLoopCount = 0
					}
					glob.LastCrash = time.Now()
					fact.FactorioBootedAt = time.Time{}
					if glob.CrashLoopCount >= 3 {
						fact.SetAutolaunch(false, true)
						msg := fmt.Sprintf("%s-%s: %s: Factorio crashed repeatedly during startup while loading. Moderator attention required, auto-start option disabled.",
							cfg.Global.GroupName, cfg.Local.Callsign, cfg.Local.Name)
						cfg.Local.Options.AutoStart = false
						cfg.WriteLCfg()

						disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, msg)
						cwlog.DoLogCW(msg)
					}
				}

				if fact.QueueFactReboot {
					if cfg.Local.Options.AutoStart {
						fact.SetAutolaunch(true, false)
					}
					fact.QueueFactReboot = false

				} else if fact.QueueReboot || glob.DoRebootCW {
					fact.DoExit(false)
					return

				} else if fact.FactAutoStart &&
					!*glob.NoAutoLaunch {

					if WithinHours() {
						launchFactorio()
					}
				}
				/* We are running normally */
			} else if fact.FactIsRunning && fact.FactorioBooted {

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
					fact.QuitFactorio(msg)
				}
			} else if fact.FactIsRunning && !fact.FactorioBooted {
				/* Startup hang watchdog: Factorio started but never became "ready" */
				if !fact.FactorioBootedAt.IsZero() && time.Since(fact.FactorioBootedAt) > constants.FactorioStartupTimeout {
					msg := fmt.Sprintf("Factorio startup exceeded %v; forcing restart.", constants.FactorioStartupTimeout)
					fact.LogCMS(cfg.Local.Channel.ChatChannel, msg)
					glob.RelaunchThrottle = 0
					fact.QuitFactorio(msg)
				}
			}
		}
	}()
}
