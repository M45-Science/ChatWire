package support

import (
	"os"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func TestHandleDesyncQueuesFactorioRebootAfterHour(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	cfg.Local.Channel.ChatChannel = "test-channel"
	glob.Uptime = time.Now().Add(-2 * time.Hour)
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

	cfg.Local.Channel.ChatChannel = "test-channel"
	glob.Uptime = time.Now().Add(-30 * time.Minute)
	fact.QueueFactReboot = false

	input := &handleData{noTimecode: "Info X DesyncedWaitingForMap Y"}
	if !handleDesync(input) {
		t.Fatalf("expected desync to be handled")
	}
	if fact.QueueFactReboot {
		t.Fatalf("did not expect Factorio reboot to be queued")
	}
}
