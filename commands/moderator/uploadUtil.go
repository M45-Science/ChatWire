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
		fact.QueueFactReboot = false  //Skip queued reboot
		fact.DoUpdateFactorio = false //Skip queued updates

		fact.SetAutolaunch(false, false)
		fact.QuitFactorio(msg)
		fact.WaitFactQuit(false)
	}
}
