package moderator

import (
	"ChatWire/fact"
	"sync"
)

const (
	saveGameName       = "save-game"
	MaxModSettingsSize = 1024 * 1024 //1MB
	MaxModListSize     = 1024 * 1024 //1MB
)

var (
	uploadLock                           sync.Mutex
	foundOption, foundSave, foundModList bool
)

func stopWaitFact(msg string) {
	if fact.FactorioBooted || fact.FactIsRunning {
		fact.QueueReboot = false      //Skip queued reboot
		fact.QueueFactReboot = false  //Skip queued reboot
		fact.DoUpdateFactorio = false //Skip queued updates

		fact.SetAutolaunch(false, false)
		fact.QuitFactorio(msg)
		fact.WaitFactQuit(false)
	}
}
