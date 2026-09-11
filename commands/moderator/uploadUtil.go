package moderator

import (
	"ChatWire/fact"
	"sync"
)

const (
	saveGameName = "save-game"
)

var (
	uploadLock                           sync.Mutex
	foundOption, foundSave, foundModList bool
)

func stopWaitFact(msg string) error {
	fact.SetAutolaunch(false, false)
	if fact.GetLifecycleState().Phase == fact.LifecycleStopped {
		return nil
	}
	return fact.StopFactorioAndWait(msg)
}
