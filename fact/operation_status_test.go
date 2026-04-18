package fact

import (
	"testing"
	"time"
)

func resetOperationStatusTestState() {
	operationStatusLock.Lock()
	operationStatus = operationStatusState{}
	operationStatusLock.Unlock()
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
