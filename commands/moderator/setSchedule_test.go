package moderator

import (
	"ChatWire/cfg"
	"strings"
	"testing"
	"time"
)

func TestParseResetDate(t *testing.T) {
	orig := cfg.Local.Options.NextReset
	defer func() { cfg.Local.Options.NextReset = orig }()

	future := time.Now().UTC().Add(time.Hour)
	out := parseResetDate(future.Format("2006-01-02 15-04-05"))
	if !strings.HasPrefix(out, "Date accepted:") {
		t.Fatalf("unexpected output %q", out)
	}
	if !cfg.Local.Options.NextReset.Equal(future.Round(time.Second)) && cfg.Local.Options.NextReset.Sub(future) > time.Second {
		t.Fatalf("reset time not set")
	}

	past := time.Now().UTC().Add(-time.Hour)
	out = parseResetDate(past.Format("2006-01-02 15-04-05"))
	if out != "That date is in the past, rejecting." {
		t.Fatalf("unexpected output %q", out)
	}

	out = parseResetDate("bad format")
	if out != "Unable to parse date provided. Format is 'YYYY-MM-DD HH-MM-SS' (24-hour UTC)" {
		t.Fatalf("unexpected output %q", out)
	}
}
