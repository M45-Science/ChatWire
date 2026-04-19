package glob

import (
	"context"
	"sync"
	"sync/atomic"
)

var (
	serverRunningState atomic.Bool
	noResponseState    atomic.Int64

	runtimeMu     sync.RWMutex
	runtimeCtx    context.Context
	runtimeCancel context.CancelFunc
)

func init() {
	serverRunningState.Store(true)
	runtimeCtx, runtimeCancel = context.WithCancel(context.Background())
}

func ServerRunning() bool {
	return serverRunningState.Load()
}

func SetServerRunning(running bool) {
	prev := serverRunningState.Swap(running)
	if running {
		if !prev {
			runtimeMu.Lock()
			runtimeCtx, runtimeCancel = context.WithCancel(context.Background())
			runtimeMu.Unlock()
		}
		return
	}

	runtimeMu.RLock()
	cancel := runtimeCancel
	runtimeMu.RUnlock()
	if cancel != nil {
		cancel()
	}
}

func RuntimeContext() context.Context {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return runtimeCtx
}

func GetNoResponseCount() int {
	return int(noResponseState.Load())
}

func SetNoResponseCount(v int) {
	NoResponseCount = v
	noResponseState.Store(int64(v))
}

func IncrementNoResponseCount() int {
	next := int(noResponseState.Add(1))
	NoResponseCount = next
	return next
}

func ResetNoResponseCount() {
	SetNoResponseCount(0)
}
