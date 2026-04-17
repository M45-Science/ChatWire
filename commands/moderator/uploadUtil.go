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

func stopWaitFact(msg string) {
	if fact.FactorioBooted || fact.FactIsRunning {
		fact.DoUpdateFactorio = false //Skip queued updates

		fact.SetAutolaunch(false, false)
		_ = fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionStop, Reason: msg})
		fact.WaitFactQuit(false)
	}
}
