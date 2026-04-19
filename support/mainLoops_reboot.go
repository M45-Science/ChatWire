package support

import (
	"fmt"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
)

func startQueuedRebootLoop() {
	/************************************
	 * Reboot if queued, when server empty
	 ************************************/
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for glob.ServerRunning() {
			<-ticker.C

			if fact.FactIsRunning && fact.FactorioBooted && fact.NumPlayersCurrent() == 0 && fact.UpdateInProgress() {
				cwlog.DoLogCW("Stopping Factorio for update.")
				_ = fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionStop, Reason: "Updating Factorio."})
			}
		}
	}()
}

func startUpdateNudgeLoop() {
	/*******************************************
	 * Bug players if there is an pending update
	 *******************************************/
	go func() {

		for glob.ServerRunning() {

			if fact.FactIsRunning && fact.FactorioBooted && fact.UpdateInProgress() {
				if fact.NumPlayersCurrent() > 0 {
					/* Warn players */
					if glob.UpdateWarnCounter < glob.UpdateGraceMinutes {
						msg := fmt.Sprintf("(SYSTEM) Factorio update waiting %v. Please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!",
							fact.NewVersion, glob.UpdateGraceMinutes-glob.UpdateWarnCounter)

						if fact.NewVersion == constants.Unknown {
							msg = fmt.Sprintf("(SYSTEM) Factorio update waiting. Please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!",
								glob.UpdateGraceMinutes-glob.UpdateWarnCounter)
						}
						fact.CMS(cfg.Local.Channel.ChatChannel, msg)
						fact.FactChat(fact.AddFactColor("red", msg))
						fact.FactChat(fact.AddFactColor("cyan", msg))
						fact.FactChat(fact.AddFactColor("black", msg))
					}

					/* Reboot anyway */
					if glob.UpdateWarnCounter > glob.UpdateGraceMinutes {
						msg := "(SYSTEM) Rebooting for Factorio update!"
						fact.FactChat(msg)
						glob.UpdateWarnCounter = 0
						_ = fact.SubmitLifecycleRequest(fact.Request{
							Kind:   fact.ActionStop,
							Reason: "Rebooting for Factorio update: " + fact.NewVersion,
						})
					}
					glob.UpdateWarnCounter = (glob.UpdateWarnCounter + 1)

					time.Sleep(time.Minute)
				} else {
					glob.UpdateWarnCounter = 0
					_ = fact.SubmitLifecycleRequest(fact.Request{
						Kind:   fact.ActionStop,
						Reason: "Rebooting for Factorio update: " + fact.NewVersion,
					})
					time.Sleep(time.Second * 5)
				}
			} else {
				time.Sleep(time.Second * 5)
			}
		}
	}()
}
