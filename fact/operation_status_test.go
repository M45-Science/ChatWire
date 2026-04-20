package fact

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/glob"
)

func resetOperationStatusTestState() {
	operationStatusLock.Lock()
	operationStatus = operationStatusState{}
	operationStatusLock.Unlock()
	glob.ResetUpdateMessage()
	operationDeleteMessage = func(channelID, messageID string) error {
		return nil
	}
}

func TestCancelOperationDelayedProgressInvalidatesPendingReminder(t *testing.T) {
	resetOperationStatusTestState()

	token := BeginOperation("Resetting map.", "Resetting map.")
	UpdateOperationProgressDelayedWithReminder(token, "Resetting map.", "Factorio is saving the map.", "Factorio is still saving the map.", 0, time.Hour)

	operationStatusLock.Lock()
	delayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	CancelOperationDelayedProgress(token)

	if emitScheduledOperationProgress(token, "Resetting map.", "Factorio is saving the map.", 0, delayID) {
		t.Fatal("expected canceled delayed reminder to be rejected")
	}

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.lastProgressKey != "" {
		t.Fatalf("expected no progress key after canceled reminder, got %q", operationStatus.lastProgressKey)
	}
}

func TestNewDelayedProgressReplacesOlderPendingReminder(t *testing.T) {
	resetOperationStatusTestState()

	token := BeginOperation("Starting Factorio.", "Starting Factorio.")
	UpdateOperationProgressDelayedWithReminder(token, "Starting Factorio.", "Factorio is loading mods.", "Factorio is continuing to load mods.", 0, time.Hour)

	operationStatusLock.Lock()
	oldDelayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	UpdateOperationProgressDelayedWithReminder(token, "Starting Factorio.", "Factorio is loading the map.", "Factorio is still loading the map.", 0, time.Hour)

	operationStatusLock.Lock()
	newDelayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	if newDelayID == oldDelayID {
		t.Fatal("expected a new delayed progress id for the replacement step")
	}
	if emitScheduledOperationProgress(token, "Starting Factorio.", "Factorio is loading mods.", 0, oldDelayID) {
		t.Fatal("expected older delayed reminder to be rejected after a newer step replaced it")
	}
	if !emitScheduledOperationProgress(token, "Starting Factorio.", "Factorio is loading the map.", 0, newDelayID) {
		t.Fatal("expected current delayed reminder to be accepted")
	}

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.description != "Factorio is loading the map." {
		t.Fatalf("expected latest progress description to win, got %q", operationStatus.description)
	}
}

func TestDelayedReminderUsesRepeatWordingAfterFirstEmission(t *testing.T) {
	resetOperationStatusTestState()

	token := BeginOperation("Starting Factorio.", "Starting Factorio.")
	UpdateOperationProgressDelayedWithReminder(token, "Starting Factorio.", "Factorio is loading mods.", "Factorio is continuing to load mods.", 0, time.Hour)

	operationStatusLock.Lock()
	delayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	if !emitScheduledOperationProgress(token, "Starting Factorio.", "Factorio is loading mods.", 0, delayID) {
		t.Fatal("expected first delayed reminder to be accepted")
	}

	operationStatusLock.Lock()
	if operationStatus.description != "Factorio is loading mods." {
		operationStatusLock.Unlock()
		t.Fatalf("expected first emission to use initial wording, got %q", operationStatus.description)
	}
	operationStatus.lastProgressUpdateAt = time.Now().Add(-operationProgressThrottle - time.Millisecond)
	operationStatusLock.Unlock()

	if !emitScheduledOperationProgress(token, "Starting Factorio.", "Factorio is continuing to load mods.", 0, delayID) {
		t.Fatal("expected repeat delayed reminder to be accepted")
	}

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.description != "Factorio is continuing to load mods." {
		t.Fatalf("expected repeat emission to use reminder wording, got %q", operationStatus.description)
	}
}

func TestFinalizeOperationRejectsOlderPendingReminders(t *testing.T) {
	resetOperationStatusTestState()

	token := BeginOperation("Starting Factorio.", "Starting Factorio.")
	UpdateOperationProgressDelayedWithReminder(token, "Starting Factorio.", "Factorio is loading the map.", "Factorio is still loading the map.", 0, time.Hour)

	operationStatusLock.Lock()
	delayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	CompleteOperation(token, "", "", 0)

	if emitScheduledOperationProgress(token, "Starting Factorio.", "Factorio is loading the map.", 0, delayID) {
		t.Fatal("expected finalized operation to reject stale delayed reminder")
	}

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.token != "" {
		t.Fatalf("expected finalized operation state to be cleared, got token %q", operationStatus.token)
	}
}

func TestFinalizeOperationDeletesTrackedOptionalMessages(t *testing.T) {
	resetOperationStatusTestState()

	token := "op-test"
	deleted := make([]string, 0, 2)
	operationDeleteMessage = func(channelID, messageID string) error {
		deleted = append(deleted, channelID+"/"+messageID)
		return nil
	}

	operationStatusLock.Lock()
	operationStatus = operationStatusState{
		token:     token,
		startedAt: time.Now().Add(-time.Minute),
		announced: true,
		optionalMessages: []operationMessageRef{
			{channelID: "c1", messageID: "m1"},
			{channelID: "c1", messageID: "m2"},
		},
	}
	operationStatusLock.Unlock()
	glob.SetUpdateMessage(&discordgo.Message{ChannelID: "c1", ID: "m2"})

	CompleteOperation(token, "", "", 0)

	if len(deleted) != 2 {
		t.Fatalf("expected 2 optional messages to be deleted, got %v", deleted)
	}
	if glob.GetUpdateMessage() != nil {
		t.Fatal("expected tracked update message to be cleared after deletion")
	}
}

func TestImmediateUpdateDeletesTrackedOptionalMessages(t *testing.T) {
	resetOperationStatusTestState()

	token := "op-test"
	deleted := make([]string, 0, 1)
	operationDeleteMessage = func(channelID, messageID string) error {
		deleted = append(deleted, channelID+"/"+messageID)
		return nil
	}

	operationStatusLock.Lock()
	operationStatus = operationStatusState{
		token:     token,
		startedAt: time.Now().Add(-time.Minute),
		announced: true,
		optionalMessages: []operationMessageRef{
			{channelID: "c1", messageID: "m1"},
		},
	}
	operationStatusLock.Unlock()
	glob.SetUpdateMessage(&discordgo.Message{ChannelID: "c1", ID: "m1"})

	UpdateOperation(token, "Starting Factorio.", "Factorio is loading mods.", 0)

	if len(deleted) != 1 || deleted[0] != "c1/m1" {
		t.Fatalf("expected optional message to be deleted, got %v", deleted)
	}
	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if len(operationStatus.optionalMessages) != 0 {
		t.Fatalf("expected optional message tracking to be cleared, got %v", operationStatus.optionalMessages)
	}
}

func TestImmediateUpdateInvalidatesQueuedDelayedReminder(t *testing.T) {
	resetOperationStatusTestState()

	token := BeginOperation("Starting Factorio.", "Starting Factorio.")
	UpdateOperationProgressDelayedWithReminder(token, "Starting Factorio.", StatusLoadingMap(""), StatusLoadingMapStill(""), 0, time.Hour)

	operationStatusLock.Lock()
	delayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	AnnounceOperationNow(token, "Starting Factorio", StatusStartingFactorio(), 0)

	if emitScheduledOperationProgress(token, "Starting Factorio.", StatusLoadingMap(""), 0, delayID) {
		t.Fatal("expected immediate update to invalidate queued delayed reminder")
	}

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.description != StatusStartingFactorio() {
		t.Fatalf("expected immediate status to remain current, got %q", operationStatus.description)
	}
}

func TestNextOptionalProgressDelayWaitsForAnnouncementWindow(t *testing.T) {
	delay := nextOptionalProgressDelay(time.Now().Add(-time.Second), false, 5*time.Second)
	expected := 5*time.Second + operationAnnounceDelay - time.Second
	if delay < expected-50*time.Millisecond || delay > expected+50*time.Millisecond {
		t.Fatalf("expected delay near %v, got %v", expected, delay)
	}
}

func TestNextOptionalProgressDelayUsesStepDelayAfterAnnouncement(t *testing.T) {
	delay := nextOptionalProgressDelay(time.Now().Add(-operationAnnounceDelay-time.Second), true, 5*time.Second)
	if delay != 5*time.Second {
		t.Fatalf("expected announced step delay to remain 5s, got %v", delay)
	}
}

func TestAnnouncePendingOperationUsesLatestQueuedDescription(t *testing.T) {
	resetOperationStatusTestState()

	token := BeginOperation("Starting Factorio", StatusStartingFactorio())
	UpdateOperation(token, "Starting Factorio", StatusLoadingMods(""), 0)

	operationStatusLock.Lock()
	operationStatus.startedAt = time.Now().Add(-operationAnnounceDelay - time.Millisecond)
	operationStatusLock.Unlock()

	AnnouncePendingOperation(token, 0)

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if !operationStatus.announced {
		t.Fatal("expected pending operation to be announced")
	}
	if operationStatus.description != StatusLoadingMods("") {
		t.Fatalf("expected latest queued description to be announced, got %q", operationStatus.description)
	}
}

func TestSuppressPendingOperationAnnouncementBlocksInitialAnnouncement(t *testing.T) {
	resetOperationStatusTestState()

	token := BeginOperation("Mod Updates", "Checking for mod updates.")
	SuppressPendingOperationAnnouncement(token)

	operationStatusLock.Lock()
	operationStatus.startedAt = time.Now().Add(-operationAnnounceDelay - time.Millisecond)
	operationStatusLock.Unlock()

	AnnouncePendingOperation(token, 0)

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if operationStatus.announced {
		t.Fatal("expected suppressed pending operation to remain unannounced")
	}
}
