package support

import (
	"strings"
	"testing"
	"time"

	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

func TestRequestFactorioStatusUsesNativeTimeUntilSoftModDetected(t *testing.T) {
	originalVersion := glob.SoftModVersion
	w := &testWriteCloser{}
	fact.SetFactorioPipe(w, 0)
	t.Cleanup(func() {
		glob.SoftModVersion = originalVersion
		fact.SetFactorioPipe(nil, 0)
	})

	glob.SoftModVersion = constants.Unknown
	requestFactorioStatus()
	if command, _ := capturedChatWireRequest(t, w.String()); command != "hello" {
		t.Fatalf("unknown SoftMod probe command = %q, want hello", command)
	}
	if !strings.Contains(w.String(), "\n/time\n") {
		t.Fatalf("unknown SoftMod probe omitted native /time fallback: %q", w.String())
	}

	w.Reset()
	glob.SoftModVersion = "test"
	requestFactorioStatus()
	if command, _ := capturedChatWireRequest(t, w.String()); command != "status" {
		t.Fatalf("detected SoftMod probe command = %q, want status", command)
	}
	if strings.Contains(w.String(), "/time") {
		t.Fatalf("detected SoftMod probe unexpectedly used native /time: %q", w.String())
	}
}

func TestWatchdogUsesSlowerPollingWhileIdle(t *testing.T) {
	oldRunning := fact.FactIsRunning
	oldBooted := fact.FactorioBooted
	oldPlayers := fact.NumPlayersCurrent()
	oldPausedTicks := fact.PausedTicks
	t.Cleanup(func() {
		fact.FactIsRunning = oldRunning
		fact.FactorioBooted = oldBooted
		fact.SetNumPlayers(oldPlayers)
		fact.PausedTicks = oldPausedTicks
	})

	fact.FactIsRunning = false
	fact.FactorioBooted = false
	fact.SetNumPlayers(0)
	fact.PausedTicks = 0
	if got := watchdogPollInterval(); got != watchdogIdleInterval {
		t.Fatalf("stopped poll interval = %v, want %v", got, watchdogIdleInterval)
	}

	fact.FactIsRunning = true
	fact.FactorioBooted = true
	if got := watchdogPollInterval(); got != watchdogIdleInterval {
		t.Fatalf("empty-server poll interval = %v, want %v", got, watchdogIdleInterval)
	}

	fact.SetNumPlayers(1)
	if got := watchdogPollInterval(); got != constants.WatchdogInterval {
		t.Fatalf("active-server poll interval = %v, want %v", got, constants.WatchdogInterval)
	}

	fact.PausedTicks = constants.PauseThresh + 1
	if got := watchdogPollInterval(); got != watchdogIdleInterval {
		t.Fatalf("paused-server poll interval = %v, want %v", got, watchdogIdleInterval)
	}
}

func TestWatchdogFailureThresholdPreservesTwoMinuteTimeout(t *testing.T) {
	tests := []struct {
		interval time.Duration
		want     int
	}{
		{interval: time.Second, want: 120},
		{interval: 15 * time.Second, want: 8},
		{interval: 0, want: 1},
	}
	for _, tc := range tests {
		if got := watchdogFailureThreshold(tc.interval); got != tc.want {
			t.Fatalf("interval %v threshold = %d, want %d", tc.interval, got, tc.want)
		}
	}
}
