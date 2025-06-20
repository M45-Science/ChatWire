package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test callback triggers when a watched file is created and then modified.
func TestWatchCreationEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	// Speed up retry loop on stat failures.
	ErrSleepDuration = 10 * time.Millisecond

	running := true
	done := make(chan struct{})
	go Watch(path, 5*time.Millisecond, &running, func() { close(done) })

	// Give watcher time to start and attempt the first stat.
	time.Sleep(20 * time.Millisecond)

	// Create the file.
	if err := os.WriteFile(path, []byte("first"), 0o644); err != nil {
		t.Fatalf("creating file: %v", err)
	}

	// Modify it to trigger the watcher.
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(path, []byte("second"), 0o644); err != nil {
		t.Fatalf("modifying file: %v", err)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("callback not triggered on creation")
	}
	running = false
}

// Test callback triggers when a watched file is modified.
func TestWatchModificationEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	if err := os.WriteFile(path, []byte("initial"), 0o644); err != nil {
		t.Fatalf("creating file: %v", err)
	}

	ErrSleepDuration = 10 * time.Millisecond

	running := true
	done := make(chan struct{})
	go Watch(path, 5*time.Millisecond, &running, func() { close(done) })

	// Allow watcher to record initial state.
	time.Sleep(20 * time.Millisecond)

	// Modify the file.
	if err := os.WriteFile(path, []byte("changed"), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("callback not triggered on modification")
	}
	running = false
}
