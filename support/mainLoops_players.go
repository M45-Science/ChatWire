package support

import (
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

func startChannelNameLoop() {
	/****************************
	* Refresh channel names
	****************************/
	go func() {

		for glob.ServerRunning {
			if !isIdle() {
				fact.UpdateChannelName()
				fact.DoUpdateChannelName()
			}

			time.Sleep(time.Duration(cfg.Local.Options.PlayerPollIntervalSec) * time.Second)
		}
	}()
}

func startOnlinePollLoop() {
	/**********************************
	* Poll online players once in a while
	**********************************/
	go func() {
		for {
			if isIdle() {
				time.Sleep(time.Duration(cfg.Local.Options.PlayerPollIntervalSec) * time.Second)
				continue
			}
			//Game booted
			if fact.FactIsRunning && fact.FactorioBooted {

				//Game isn't paused
				if fact.PausedTicks <= constants.PauseThresh {
					fact.WriteFact(glob.OnlineCommand)
				}
			}
			time.Sleep(time.Duration(cfg.Local.Options.PlayerPollIntervalSec) * time.Second)
		}
	}()
}

func startPlayerTimeLoop() {
	/****************************/
	/* Update player time       */
	/****************************/
	go func() {
		for glob.ServerRunning {
			if isIdle() {
				time.Sleep(time.Minute)
				continue
			}
			now := time.Now()
			updated := false
			glob.PlayerListLock.Lock() //Lock
			for _, p := range glob.PlayerList {
				if now.Sub(fact.ExpandTime(p.LastSeen)) <= time.Minute {
					p.Minutes++
					updated = true
				}
			}
			glob.PlayerListLock.Unlock() //Unlock
			if updated {
				fact.SetPlayerStatsDirty()
			}
			time.Sleep(time.Minute)
		}
	}()
}
