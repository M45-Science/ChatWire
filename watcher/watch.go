package watcher

import (
	"os"
	"time"

	"ChatWire/cwlog"
)

// Watch monitors a file and invokes cb whenever the file is modified.
// The loop stops when running is nil or *running becomes false.
func Watch(path string, interval time.Duration, running *bool, cb func()) {
	for running == nil || *running {
		initial, err := os.Stat(path)
		if err != nil {
			cwlog.DoLogCW("watcher: initial stat error on %s: %v", path, err)
			time.Sleep(time.Minute)
			continue
		}

		time.Sleep(interval)
		for running == nil || *running {
			time.Sleep(interval)

			stat, err := os.Stat(path)
			if err != nil {
				cwlog.DoLogCW("watcher: stat error on %s: %v", path, err)
				time.Sleep(time.Minute)
				break
			}
			if stat.Size() != initial.Size() || stat.ModTime() != initial.ModTime() {
				cb()
				break
			}
		}
	}
}
