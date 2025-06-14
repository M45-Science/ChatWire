package support

import (
	"ChatWire/cfg"
	"testing"
	"time"
)

func TestWithinHours(t *testing.T) {
	cur := time.Now().UTC().Hour()

	cfg.Local.Options.PlayHourEnable = false
	if !WithinHours() {
		t.Fatalf("expected true when disabled")
	}

	cfg.Local.Options.PlayHourEnable = true
	cfg.Local.Options.PlayStartHour = (cur + 23) % 24
	cfg.Local.Options.PlayEndHour = (cur + 1) % 24
	if !WithinHours() {
		t.Fatalf("expected true within window")
	}

	cfg.Local.Options.PlayStartHour = (cur + 1) % 24
	cfg.Local.Options.PlayEndHour = (cur + 2) % 24
	if WithinHours() {
		t.Fatalf("expected false outside window")
	}

	cfg.Local.Options.PlayStartHour = (cur + 2) % 24
	cfg.Local.Options.PlayEndHour = (cur + 1) % 24
	if !WithinHours() {
		t.Fatalf("expected true across midnight")
	}

	cfg.Local.Options.PlayStartHour = (cur + 1) % 24
	cfg.Local.Options.PlayEndHour = (cur + 23) % 24
	if WithinHours() {
		t.Fatalf("expected false outside wrapped window")
	}
}
