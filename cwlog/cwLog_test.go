package cwlog

import (
	"testing"
	"time"
)

func TestDurationUntilNextUTCMidnight(t *testing.T) {
	now := time.Date(2026, time.September, 12, 23, 59, 30, 0, time.FixedZone("MDT", -6*60*60))
	// 23:59:30 MDT is 05:59:30 UTC, so the next UTC midnight is 18:00 MDT.
	if got, want := durationUntilNextUTCMidnight(now), 18*time.Hour+30*time.Second; got != want {
		t.Fatalf("duration = %v, want %v", got, want)
	}
}
