package support

import (
	"os"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
)

func TestHandleDesyncQueuesFactorioRebootAfterFifteenMinutes(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	origBootedAt := fact.FactorioBootedAt
	t.Cleanup(func() {
		fact.FactorioBootedAt = origBootedAt
	})

	cfg.Local.Channel.ChatChannel = "test-channel"
	fact.FactorioBootedAt = time.Now().Add(-20 * time.Minute)
	fact.QueueFactReboot = false

	input := &handleData{noTimecode: "Info X DesyncedWaitingForMap Y"}
	if !handleDesync(input) {
		t.Fatalf("expected desync to be handled")
	}
	if !fact.QueueFactReboot {
		t.Fatalf("expected Factorio reboot to be queued")
	}

	deadline := time.NewTimer(250 * time.Millisecond)
	defer deadline.Stop()
	foundQueuedMsg := false
	for !foundQueuedMsg {
		select {
		case msg := <-disc.CMSChan:
			if strings.Contains(msg.Text, "reboot queued") {
				foundQueuedMsg = true
			}
		case <-deadline.C:
			t.Fatalf("expected a queued-reboot message on CMSChan")
		}
	}
}

func TestHandleDesyncDoesNotQueueBeforeHour(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	origBootedAt := fact.FactorioBootedAt
	t.Cleanup(func() {
		fact.FactorioBootedAt = origBootedAt
	})

	cfg.Local.Channel.ChatChannel = "test-channel"
	fact.FactorioBootedAt = time.Now().Add(-5 * time.Minute)
	fact.QueueFactReboot = false

	input := &handleData{noTimecode: "Info X DesyncedWaitingForMap Y"}
	if !handleDesync(input) {
		t.Fatalf("expected desync to be handled")
	}
	if fact.QueueFactReboot {
		t.Fatalf("did not expect Factorio reboot to be queued")
	}
}
