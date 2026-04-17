package fact

import "sync/atomic"

var (
	gameLineChState   atomic.Pointer[chan string]
	updateState       atomic.Bool
	modOperationState atomic.Bool
	autoStartState    atomic.Bool
	numPlayersState   atomic.Int64
)

func SetGameLineCh(ch chan string) {
	gameLineCh = ch
	if ch == nil {
		gameLineChState.Store(nil)
		return
	}
	chCopy := ch
	gameLineChState.Store(&chCopy)
}

func GameLineChCurrent() chan string {
	ptr := gameLineChState.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}

func SetUpdateInProgress(v bool) {
	DoUpdateFactorio = v
	updateState.Store(v)
}

func UpdateInProgress() bool {
	return updateState.Load()
}

func SetModOperationInProgress(v bool) {
	DoModOperation = v
	modOperationState.Store(v)
}

func ModOperationInProgress() bool {
	return modOperationState.Load()
}

func setAutostartEnabled(v bool) {
	FactAutoStart = v
	autoStartState.Store(v)
}

func AutostartEnabled() bool {
	return autoStartState.Load()
}

func SetNumPlayers(v int) {
	NumPlayers = v
	numPlayersState.Store(int64(v))
}

func NumPlayersCurrent() int {
	return int(numPlayersState.Load())
}
