package support

import (
	"testing"
	"time"

	"ChatWire/fact"
	"ChatWire/glob"
)

func TestHandleChatSwitchesChannels(t *testing.T) {
	origRunning := glob.ServerRunning()
	origCh := fact.GameLineChCurrent()
	origNoResponse := glob.GetNoResponseCount()
	defer func() {
		glob.SetServerRunning(origRunning)
		fact.SetGameLineCh(origCh)
		glob.SetNoResponseCount(origNoResponse)
	}()

	glob.SetServerRunning(true)
	glob.SetNoResponseCount(123)

	ch1 := make(chan string)
	fact.SetGameLineCh(ch1)

	done := make(chan struct{})
	go func() {
		HandleChat()
		close(done)
	}()

	// Give HandleChat time to start and block on the initial channel.
	time.Sleep(50 * time.Millisecond)

	ch2 := make(chan string, 1)
	fact.SetGameLineCh(ch2)
	ch2 <- "foo"

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if glob.GetNoResponseCount() == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if glob.GetNoResponseCount() != 0 {
		glob.SetServerRunning(false)
		t.Fatalf("HandleChat did not switch to the new channel (NoResponseCount=%v)", glob.GetNoResponseCount())
	}

	glob.SetServerRunning(false)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("HandleChat did not stop after ServerRunning=false")
	}
}
