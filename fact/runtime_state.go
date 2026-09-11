package fact

import (
	"io"
	"sync/atomic"
)

var (
	gameLineChState   atomic.Pointer[gameOutput]
	updateState       atomic.Bool
	modOperationState atomic.Bool
	autoStartState    atomic.Bool
	numPlayersState   atomic.Int64
)

type gameOutput struct {
	lines      chan string
	generation uint64
}

func SetGameLineCh(ch chan string, generation uint64) {
	if ch == nil {
		gameLineChState.Store(nil)
		return
	}
	gameLineChState.Store(&gameOutput{lines: ch, generation: generation})
}

func GameOutputCurrent() (chan string, uint64) {
	output := gameLineChState.Load()
	if output == nil {
		return nil, 0
	}
	return output.lines, output.generation
}

func GameLineChCurrent() chan string {
	lines, _ := GameOutputCurrent()
	return lines
}

func SetFactorioPipe(pipe io.WriteCloser, generation uint64) {
	PipeLock.Lock()
	Pipe, pipeGeneration = pipe, generation
	PipeLock.Unlock()
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
