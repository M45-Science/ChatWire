package fact

import (
	"errors"
	"testing"
	"time"
)

func TestCloseDiscordSessionReturnsCloseError(t *testing.T) {
	want := errors.New("close failed")
	got := closeDiscordSession(func() error { return want }, time.Second)
	if !errors.Is(got, want) {
		t.Fatalf("closeDiscordSession() error = %v, want %v", got, want)
	}
}

func TestCloseDiscordSessionTimesOut(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	started := time.Now()
	got := closeDiscordSession(func() error {
		<-release
		return nil
	}, 20*time.Millisecond)
	if !errors.Is(got, errDiscordCloseTimeout) {
		t.Fatalf("closeDiscordSession() error = %v, want %v", got, errDiscordCloseTimeout)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("closeDiscordSession() took %v; timeout was not bounded", elapsed)
	}
}
