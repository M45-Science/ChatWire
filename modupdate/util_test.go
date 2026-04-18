package modupdate

import "testing"

func TestGetCompletedDownloadCount(t *testing.T) {
	downloads := []downloadData{
		{Name: "a", Complete: true},
		{Name: "b", Complete: false},
		{Name: "c", Complete: true},
	}

	if got := getCompletedDownloadCount(downloads); got != 2 {
		t.Fatalf("expected 2 completed downloads, got %d", got)
	}
}
