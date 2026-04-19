package watcher

import (
	"context"
	"os"
	"time"

	"ChatWire/cwlog"
)

// ErrSleepDuration controls how long Watch waits after encountering an
// error while stat'ing the file. It is exported so tests can shorten the
// delay from the default minute.
var ErrSleepDuration = time.Minute

// Watch monitors a file and invokes cb whenever the file is modified.
// The loop stops when ctx is canceled.
func Watch(path string, interval time.Duration, ctx context.Context, cb func()) {
	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		initial, err := os.Stat(path)
		if err != nil {
			cwlog.DoLogCW("watcher: initial stat error on %s: %v", path, err)
			if !sleepWithContext(ctx, ErrSleepDuration) {
				return
			}
			continue
		}

		if !sleepWithContext(ctx, interval) {
			return
		}
		for {
			if !sleepWithContext(ctx, interval) {
				return
			}

			stat, err := os.Stat(path)
			if err != nil {
				cwlog.DoLogCW("watcher: stat error on %s: %v", path, err)
				if !sleepWithContext(ctx, ErrSleepDuration) {
					return
				}
				break
			}
			if stat.Size() != initial.Size() || stat.ModTime() != initial.ModTime() {
				cb()
				break
			}
		}
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if ctx == nil {
		time.Sleep(d)
		return true
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
