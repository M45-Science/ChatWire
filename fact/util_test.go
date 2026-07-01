package fact

import (
	"errors"
	"io"
	"testing"
	"time"

	"ChatWire/cfg"
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

func TestMakeSteamURLDisabledByDefaultReturnsEmpty(t *testing.T) {
	oldGlobal := cfg.Global
	oldLocal := cfg.Local
	t.Cleanup(func() {
		cfg.Global = oldGlobal
		cfg.Local = oldLocal
	})

	cfg.Global.Paths.URLs.Domain = "factorio.example.com"
	cfg.Global.Paths.URLs.SteamURLDomain = ""
	cfg.Global.Paths.URLs.EnableSteamURL = false
	cfg.Local.Port = 34197

	got, ok := MakeSteamURL()
	if ok {
		t.Fatalf("MakeSteamURL configured unexpectedly: %q", got)
	}
	if got != "" {
		t.Fatalf("unexpected disabled URL: %q", got)
	}
}

func TestMakeSteamURLUsesConfiguredDomain(t *testing.T) {
	oldGlobal := cfg.Global
	oldLocal := cfg.Local
	t.Cleanup(func() {
		cfg.Global = oldGlobal
		cfg.Local = oldLocal
	})

	cfg.Global.Paths.URLs.Domain = "factorio.example.com"
	cfg.Global.Paths.URLs.SteamURLDomain = "steam.example.com"
	cfg.Global.Paths.URLs.EnableSteamURL = true
	cfg.Local.Port = 34197

	got, ok := MakeSteamURL()
	if !ok {
		t.Fatal("MakeSteamURL reported not configured")
	}

	want := "https://steam.example.com/gosteam/427520.--mp-connect%20factorio.example.com:34197"
	if got != want {
		t.Fatalf("unexpected Steam URL: got %q want %q", got, want)
	}
}
