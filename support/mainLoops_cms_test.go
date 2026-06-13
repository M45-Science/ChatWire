package support

import "testing"

func TestAppendCMSBufferLineDoesNotPrefixBlankLine(t *testing.T) {
	got := appendCMSBufferLine("", "Checking for Factorio updates.")
	if got != "Checking for Factorio updates." {
		t.Fatalf("expected first line without prefix newline, got %q", got)
	}

	got = appendCMSBufferLine(got, "Checking latest Factorio release failed.")
	want := "Checking for Factorio updates.\nChecking latest Factorio release failed."
	if got != want {
		t.Fatalf("buffer mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestAppendCMSBufferLineIgnoresBlankLines(t *testing.T) {
	got := appendCMSBufferLine("Checking for Factorio updates.", " \r")
	if got != "Checking for Factorio updates." {
		t.Fatalf("expected blank line to be ignored, got %q", got)
	}
}
