package fact

import (
	"errors"
	"io"
	"testing"
	"time"
)

type failingWriteCloser struct {
	err error
}

func (f failingWriteCloser) Write(p []byte) (int, error) {
	return 0, f.err
}

func (f failingWriteCloser) Close() error {
	return nil
}

func TestClassifyFactorioPipeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "closed pipe", err: io.ErrClosedPipe, want: "stdin-closed"},
		{name: "broken pipe", err: errors.New("write |1: broken pipe"), want: "stdin-broken-pipe"},
		{name: "generic", err: errors.New("write failed"), want: "stdin-write-failed"},
	}

	for _, tc := range tests {
		if got := classifyFactorioPipeError(tc.err); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestWriteFactBrokenPipeQueuesHealthEvent(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleProcessAlive = func() bool { return true }
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.currentGeneration = 21
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()

	PipeLock.Lock()
	Pipe = failingWriteCloser{err: errors.New("write |1: broken pipe")}
	PipeLock.Unlock()
	defer func() {
		PipeLock.Lock()
		Pipe = nil
		PipeLock.Unlock()
	}()

	WriteFact("/time")
	lm.drainAsyncEvents()

	lm.mu.Lock()
	defer lm.mu.Unlock()
	if !lm.healthRestartQueued {
		t.Fatal("expected broken pipe write to queue a health restart")
	}
}
