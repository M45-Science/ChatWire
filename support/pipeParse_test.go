package support

import (
	"testing"
	"time"

	"ChatWire/fact"
	"ChatWire/glob"
)

func TestHandleChatSwitchesChannels(t *testing.T) {
	origRunning := glob.ServerRunning
	origCh := fact.GameLineCh
	origNoResponse := glob.NoResponseCount
	defer func() {
		glob.ServerRunning = origRunning
		fact.GameLineCh = origCh
		glob.NoResponseCount = origNoResponse
	}()

	glob.ServerRunning = true
	glob.NoResponseCount = 123

	ch1 := make(chan string)
	fact.GameLineCh = ch1

	done := make(chan struct{})
	go func() {
		HandleChat()
		close(done)
	}()

	// Give HandleChat time to start and block on the initial channel.
	time.Sleep(50 * time.Millisecond)

	ch2 := make(chan string, 1)
	fact.GameLineCh = ch2
	ch2 <- "foo"

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if glob.NoResponseCount == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if glob.NoResponseCount != 0 {
		glob.ServerRunning = false
		t.Fatalf("HandleChat did not switch to the new channel (NoResponseCount=%v)", glob.NoResponseCount)
	}

	glob.ServerRunning = false

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("HandleChat did not stop after ServerRunning=false")
	}
}

