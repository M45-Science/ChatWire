package support

import (
	"time"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/worker"
)

func startPasscodeCleanup() {
	/************************************
	 * Delete expired registration codes
	 ************************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(1 * time.Minute)

			t := time.Now()

			glob.PasswordListLock.Lock()
			for _, pass := range glob.PassList {
				if (t.Unix() - pass.Time) > constants.PassExpireSec {
					cwlog.DoLogCW("Invalidating unused registration code for player: " + disc.GetNameFromID(pass.DiscID))
					delete(glob.PassList, pass.DiscID)
				}
			}
			glob.PasswordListLock.Unlock()
		}
	}()
}

func startPlayerListSaveLoop() {
	/********************************
	 * Save database, if marked dirty
	 ********************************/
	go func() {
		time.Sleep(time.Minute)
		saveDirty := newDebounce(10*time.Second, func() {
			worker.Submit(func() {
				cwlog.DoLogCW("Database marked dirty, saving.")
				fact.WritePlayers()
			})
		})

		for glob.ServerRunning {
			<-fact.PlayerListDirtySignal()
			glob.PlayerListDirtyLock.Lock()
			wasDirty := glob.PlayerListDirty
			glob.PlayerListDirty = false
			glob.PlayerListDirtyLock.Unlock()
			if wasDirty {
				saveDirty.trigger()
			}
		}
	}()
}

func startPlayerSeenSaveLoop() {
	/************************************************************
	 * Save database (low priority) when player stats are dirty.
	 * This covers LastSeen and Minutes, and saves on a slower
	 * cadence to reduce churn.
	 ************************************************************/
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for glob.ServerRunning {
			<-ticker.C
			glob.PlayerStatsDirtyLock.Lock()
			wasDirty := glob.PlayerStatsDirty
			glob.PlayerStatsDirty = false
			glob.PlayerStatsDirtyLock.Unlock()
			if wasDirty {
				worker.Submit(func() {
					fact.WritePlayers()
				})
			}
		}
	}()
}
