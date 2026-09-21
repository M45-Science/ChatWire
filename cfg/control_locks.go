package cfg

import (
	"ChatWire/util"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

var controlLocksEnabled atomic.Bool

// EnableControlLocks enables cross-process game-file coordination on hosts using
// web control. All managed instances should enable the private control endpoint.
func EnableControlLocks() { controlLocksEnabled.Store(true) }
func LockControlResources() (func(), error) {
	if !controlLocksEnabled.Load() {
		return func() {}, nil
	}
	root, e := filepath.Abs(GetFactorioFolder())
	if e != nil {
		return nil, e
	}
	if resolved, e := filepath.EvalSymlinks(root); e == nil {
		root = resolved
	}
	if e = os.MkdirAll(root, 0755); e != nil {
		return nil, e
	}
	l, e := util.AcquireFileLock(filepath.Join(root, ".chatwire-control.lock"), "game-files", 5*time.Second)
	if e != nil {
		return nil, e
	}
	return func() { _ = l.Release() }, nil
}
