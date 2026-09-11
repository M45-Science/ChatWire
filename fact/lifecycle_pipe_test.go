package fact

import (
	"io"
	"runtime"
	"testing"
	"time"
)

type blockedErrorPipe struct{ entered, release chan struct{} }

func (w *blockedErrorPipe) Close() error { return nil }
func (w *blockedErrorPipe) Write(p []byte) (int, error) {
	close(w.entered)
	<-w.release
	return 0, io.ErrClosedPipe
}

func TestRegressionShutdownPipeErrorDoesNotDeadlock(t *testing.T) {
	resetLifecycleTestState(t)
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase, lm.currentGeneration = LifecycleRunning, 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()
	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()
	t.Cleanup(func() { lifecycleMu.Lock(); lifecycle = nil; lifecycleMu.Unlock() })
	w := &blockedErrorPipe{entered: make(chan struct{}), release: make(chan struct{})}
	SetFactorioPipe(w, 1)
	writeDone := make(chan struct{})
	go func() { WriteFact("/online"); close(writeDone) }()
	<-w.entered
	stopDone := make(chan struct{})
	go func() { lm.finalizeStopped(1, nil, false); close(stopDone) }()
	deadline := time.Now().Add(time.Second)
	for {
		if !lm.mu.TryLock() {
			break
		}
		lm.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("stop did not acquire lifecycle lock")
		}
		time.Sleep(time.Millisecond)
	}
	close(w.release)
	select {
	case <-writeDone:
	case <-time.After(time.Second):
		stack := make([]byte, 16384)
		n := runtime.Stack(stack, true)
		t.Fatalf("pipe error and process exit deadlocked:\n%s", stack[:n])
	}
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("stop did not finish")
	}
}
