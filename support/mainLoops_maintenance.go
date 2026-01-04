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

func startPanelTokenCleanup() {
	/********************************
	 * Delete expired panel tokens
	 ********************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(1 * time.Minute)

			t := time.Now()

			glob.PanelTokenLock.Lock()
			for k, tok := range glob.PanelTokens {
				if (t.Unix()-tok.Time) > constants.PassExpireSec || (t.Unix()-tok.Orig) > constants.PanelTokenLimitSec {
					delete(glob.PanelTokens, k)
				}
			}
			glob.PanelTokenLock.Unlock()
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
	/***********************************************************
	 * Save database (less often), if last seen is marked dirty
	 ***********************************************************/
	go func() {
		time.Sleep(time.Minute)
		saveSeenDirty := newDebounce(30*time.Second, func() {
			worker.Submit(func() {
				//cwlog.DoLogCW("Database last seen flagged, saving.")
				fact.WritePlayers()
			})
		})

		for glob.ServerRunning {
			<-fact.PlayerListSeenDirtySignal()
			glob.PlayerListSeenDirtyLock.Lock()
			wasDirty := glob.PlayerListSeenDirty
			glob.PlayerListSeenDirty = false
			glob.PlayerListSeenDirtyLock.Unlock()
			if wasDirty {
				saveSeenDirty.trigger()
			}
		}
	}()
}
