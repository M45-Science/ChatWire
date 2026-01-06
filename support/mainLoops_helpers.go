package support

import (
	"sync"
	"time"

	"ChatWire/constants"
	"ChatWire/fact"
)

func isIdle() bool {
	return !fact.FactIsRunning || !fact.FactorioBooted || fact.PausedTicks > constants.PauseThresh
}

type debounce struct {
	mu    sync.Mutex
	wait  time.Duration
	timer *time.Timer
	fn    func()
}

func newDebounce(wait time.Duration, fn func()) *debounce {
	return &debounce{wait: wait, fn: fn}
}

func (d *debounce) trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.wait, d.fn)
}
