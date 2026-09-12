package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileLockMetadataAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json.lock")
	lock, err := AcquireFileLock(path, "server-a", time.Second)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var metadata FileLockMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("invalid lock metadata: %v", err)
	}
	if metadata.PID != os.Getpid() || metadata.Owner != "server-a" || metadata.AcquiredAt.IsZero() {
		t.Fatalf("unexpected lock metadata: %+v", metadata)
	}

	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := AcquireFileLock(path, "server-b", time.Second)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestFileLockTimesOutWithHolderMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json.lock")
	lock, err := AcquireFileLock(path, "server-a", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()

	_, err = AcquireFileLock(path, "server-b", 75*time.Millisecond)
	if err == nil {
		t.Fatal("expected competing lock acquisition to time out")
	}
	if !strings.Contains(err.Error(), "server-a") || !strings.Contains(err.Error(), "acquired_at") {
		t.Fatalf("timeout did not identify the lock holder: %v", err)
	}
}
