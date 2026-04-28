package banlist

import (
	"strings"
	"testing"
)

func TestParseBanFileDataStandardFormat(t *testing.T) {
	got, err := parseBanFileData([]byte(`[{"username":"Alice","reason":"griefing"},{"username":"Bob"}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bans, got %d", len(got))
	}
	if got[0].UserName != "Alice" || got[0].Reason != "griefing" {
		t.Fatalf("unexpected first ban: %#v", got[0])
	}
	if got[1].UserName != "Bob" {
		t.Fatalf("unexpected second ban: %#v", got[1])
	}
}

func TestParseBanFileDataLegacyFormat(t *testing.T) {
	got, err := parseBanFileData([]byte(`["Alice","","BOB"]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bans, got %d", len(got))
	}
	if got[0].UserName != "alice" || got[1].UserName != "bob" {
		t.Fatalf("expected legacy names to be normalized, got %#v", got)
	}
}

func TestParseBanFileDataInvalid(t *testing.T) {
	_, err := parseBanFileData([]byte(`{"username":"Alice"}`))
	if err == nil {
		t.Fatal("expected invalid ban data to return an error")
	}
	if !strings.Contains(err.Error(), "standard format parse error") ||
		!strings.Contains(err.Error(), "legacy format parse error") {
		t.Fatalf("expected both parse errors, got %v", err)
	}
}
