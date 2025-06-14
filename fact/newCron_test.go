package fact

import (
	"ChatWire/cfg"
	"testing"
	"time"
)

func TestFormatResetTime(t *testing.T) {
	orig := cfg.Local.Options.NextReset
	defer func() { cfg.Local.Options.NextReset = orig }()

	cfg.Local.Options.NextReset = time.Time{}
	if got := FormatResetTime(); got != "No reset is scheduled" {
		t.Fatalf("unexpected %q", got)
	}

	tval := time.Date(2025, time.January, 2, 15, 4, 0, 0, time.UTC)
	cfg.Local.Options.NextReset = tval
	exp := tval.UTC().Format("Jan 02, 03:04PM UTC")
	if got := FormatResetTime(); got != exp {
		t.Fatalf("expected %q got %q", exp, got)
	}
}

func TestFormatResetInterval(t *testing.T) {
	orig := cfg.Local.Options.ResetInterval
	defer func() { cfg.Local.Options.ResetInterval = orig }()

	cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
	if got := FormatResetInterval(); got != "No reset interval is set" {
		t.Fatalf("unexpected %q", got)
	}

	cfg.Local.Options.ResetInterval = cfg.ResetInterval{Months: 1, Days: 2, Hours: 3}
	if got := FormatResetInterval(); got != "Every 1 month, 2 days, 3 hours" {
		t.Fatalf("unexpected %q", got)
	}
}

func TestSetResetDateAdvanceAndTimeTill(t *testing.T) {
	origNext := cfg.Local.Options.NextReset
	origInterval := cfg.Local.Options.ResetInterval
	origHour := cfg.Local.Options.ResetHour
	defer func() {
		cfg.Local.Options.NextReset = origNext
		cfg.Local.Options.ResetInterval = origInterval
		cfg.Local.Options.ResetHour = origHour
	}()

	cfg.Local.Options.ResetInterval = cfg.ResetInterval{Hours: 2}
	cfg.Local.Options.ResetHour = 0
	start := time.Now().UTC()
	SetResetDate()
	diff := cfg.Local.Options.NextReset.Sub(start.Add(2 * time.Hour))
	if diff < -time.Second || diff > time.Second {
		t.Fatalf("unexpected next reset diff %v", diff)
	}

	start = cfg.Local.Options.NextReset
	AdvanceReset()
	if !cfg.Local.Options.NextReset.Equal(start.Add(2 * time.Hour)) {
		t.Fatalf("advance failed: %v", cfg.Local.Options.NextReset.Sub(start))
	}

	cfg.Local.Options.NextReset = time.Now().UTC().Add(90 * time.Minute).Round(time.Second)
	if got := TimeTillReset(); got != "1 hour 30 minutes" {
		t.Fatalf("unexpected till reset %q", got)
	}

	cfg.Local.Options.NextReset = time.Now().UTC().Add(70 * time.Second).Round(time.Second)
	if got := TimeTillReset(); got != "1 minute" {
		t.Fatalf("unexpected till reset %q", got)
	}

	cfg.Local.Options.NextReset = time.Time{}
	if got := TimeTillReset(); got != "No reset is scheduled" {
		t.Fatalf("unexpected %q", got)
	}
}
