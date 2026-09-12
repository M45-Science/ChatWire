package fact

import (
	"testing"
	"time"
)

func TestCurrentGametimeUsesObservedUPS(t *testing.T) {
	start := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)

	ResetGametime(60)
	RecordGameTime(0, 0, 0, 10, "10 seconds", 60, start)
	RecordGameTime(0, 0, 0, 15, "15 seconds", 60, start.Add(10*time.Second))
	if got := currentGametimeAt(start.Add(12 * time.Second)); got != "16" {
		t.Fatalf("30 UPS interpolation = %q, want %q", got, "16")
	}

	ResetGametime(60)
	RecordGameTime(0, 0, 0, 10, "10 seconds", 60, start)
	RecordGameTime(0, 0, 0, 20, "20 seconds", 60, start.Add(5*time.Second))
	if got := currentGametimeAt(start.Add(7 * time.Second)); got != "24" {
		t.Fatalf("120 UPS interpolation = %q, want %q", got, "24")
	}
}

func TestCurrentGametimeMeasuresLowUPSOverRepeatedSamples(t *testing.T) {
	start := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	ResetGametime(4)
	RecordGameTime(0, 0, 0, 10, "10 seconds", 4, start)
	RecordGameTime(0, 0, 0, 10, "10 seconds", 4, start.Add(5*time.Second))
	RecordGameTime(0, 0, 0, 10, "10 seconds", 4, start.Add(10*time.Second))
	RecordGameTime(0, 0, 0, 11, "11 seconds", 4, start.Add(15*time.Second))

	if got := currentGametimeAt(start.Add(30 * time.Second)); got != "12" {
		t.Fatalf("4 UPS interpolation = %q, want %q", got, "12")
	}
}

func TestCurrentGametimeUsesAuthoritativeCorrection(t *testing.T) {
	start := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	ResetGametime(120)
	RecordGameTime(0, 0, 0, 10, "10 seconds", 120, start)
	if got := currentGametimeAt(start.Add(4 * time.Second)); got != "18" {
		t.Fatalf("pre-correction time = %q, want %q", got, "18")
	}

	RecordGameTime(0, 0, 0, 17, "17 seconds", 120, start.Add(4*time.Second))
	if got := currentGametimeAt(start.Add(4 * time.Second)); got != "17" {
		t.Fatalf("corrected time = %q, want %q", got, "17")
	}
}

func TestCurrentGametimeStopsWhilePaused(t *testing.T) {
	start := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	ResetGametime(60)
	RecordGameTime(0, 1, 2, 3, "1 minute, 2 seconds", 60, start)
	SetGameTimePaused(true)

	if got := currentGametimeAt(start.Add(time.Minute)); got != "01-02-03" {
		t.Fatalf("paused time = %q, want %q", got, "01-02-03")
	}
}

func TestRecordGameTimeReportsUnchangedSample(t *testing.T) {
	start := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	ResetGametime(60)
	if RecordGameTime(0, 0, 0, 10, "10 seconds", 60, start) {
		t.Fatal("first sample reported unchanged")
	}
	if !RecordGameTime(0, 0, 0, 10, "10 seconds", 60, start.Add(time.Second)) {
		t.Fatal("repeated sample did not report unchanged")
	}
}

func TestRecordGameTickPreservesFractionalSecondsAtLowUPS(t *testing.T) {
	start := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	ResetGametime(6)
	RecordGameTick(600, 0.1, false, start)
	RecordGameTick(690, 0.1, false, start.Add(15*time.Second))
	if got := currentGametimeAt(start.Add(30 * time.Second)); got != "13" {
		t.Fatalf("low-UPS tick interpolation = %q, want 13", got)
	}
}

func TestCurrentGametimeStringFormatsTickLikeChat(t *testing.T) {
	start := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	ResetGametime(60)
	RecordGameTick(3202801, 1, true, start)

	if got := CurrentGametimeString(); got != "14-49-40" {
		t.Fatalf("displayed tick time = %q, want %q", got, "14-49-40")
	}
}
