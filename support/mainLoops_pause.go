package support

import (
	"fmt"
	"time"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
)

func startPauseExpiryLoop() {
	/* Check for expired pauses */
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for glob.ServerRunning {
			glob.PausedLock.Lock()

			if glob.PausedForConnect {

				limit := time.Minute * 3

				now := time.Now()
				if now.Sub(glob.PausedAt) > limit {

					fact.WriteFact(
						fmt.Sprintf("/gspeed %0.2f", cfg.Local.Options.Speed))

					if glob.PausedConnectAttempt {
						msg := "Unpausing, " + glob.PausedFor + " did not finish joining within the time limit."
						fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, msg)
					}

					glob.PausedForConnect = false
					glob.PausedFor = ""
					glob.PausedConnectAttempt = false
				}
			} else {
				/* Eventually reset timers */
				if glob.PausedCount > 0 {
					now := time.Now()
					if now.Sub(glob.PausedAt) > time.Minute*30 {
						glob.PausedCount = 0
						glob.PausedAt = now
						glob.PausedFor = ""
						glob.PausedConnectAttempt = false
					}
				}
			}
			glob.PausedLock.Unlock()
			<-ticker.C
		}
	}()
}
