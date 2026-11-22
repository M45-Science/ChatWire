package cwlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ChatWire/glob"
)

func TestStartGameLogLinksOnlyTheGameLog(t *testing.T) {
	t.Chdir(t.TempDir())
	oldName, oldDesc := glob.GameLogName, glob.GameLogDesc
	glob.GameLogDesc = nil
	t.Cleanup(func() {
		if glob.GameLogDesc != nil {
			_ = glob.GameLogDesc.Close()
		}
		glob.GameLogName, glob.GameLogDesc = oldName, oldDesc
	})

	StartGameLog()

	newest := filepath.Join("log", "newest.log")
	target, err := os.Readlink(newest)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Base(glob.GameLogName); target != want {
		t.Errorf("%s points to %q, want %q", newest, target, want)
	}
	for _, name := range []string{"factorio-current.log", "factorio-previous.log"} {
		link := filepath.Join("log", name)
		if _, err := os.Lstat(link); !os.IsNotExist(err) {
			t.Errorf("expected %s to be absent, got %v", link, err)
		}
	}
}

func TestDurationUntilNextUTCMidnight(t *testing.T) {
	now := time.Date(2026, time.September, 12, 23, 59, 30, 0, time.FixedZone("MDT", -6*60*60))
	// 23:59:30 MDT is 05:59:30 UTC, so the next UTC midnight is 18:00 MDT.
	if got, want := durationUntilNextUTCMidnight(now), 18*time.Hour+30*time.Second; got != want {
		t.Fatalf("duration = %v, want %v", got, want)
	}
}
