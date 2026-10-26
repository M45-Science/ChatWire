package user

import "testing"

func TestMapPreviewRCONReply(t *testing.T) {
	const url = "https://mapgen.m45sci.xyz/#m=YWJj"
	got, err := mapPreviewURLFromReply("\r\n" + url + "\n")
	if err != nil || got != url {
		t.Fatalf("url=%q err=%v", got, err)
	}
	for _, reply := range []string{"", "Unknown command: map-preview", "https://other.example/#m=YWJj", "https://mapgen.m45sci.xyz/#m=", url + "\n[truncated]"} {
		if got, err := mapPreviewURLFromReply(reply); err == nil || got != "" {
			t.Fatalf("accepted invalid RCON reply %q", reply)
		}
	}
}
