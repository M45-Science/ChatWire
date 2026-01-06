package fact

import (
	"testing"

	"ChatWire/glob"
)

func TestUpdateSeenMarksStatsDirtyOnly(t *testing.T) {
	glob.PlayerList = make(map[string]*glob.PlayerData)
	glob.PlayerStatsDirty = false
	glob.PlayerListDirty = false

	glob.PlayerList["alice"] = &glob.PlayerData{Name: "alice"}
	UpdateSeen("alice")

	if !glob.PlayerStatsDirty {
		t.Fatalf("expected PlayerStatsDirty to be set")
	}
	if glob.PlayerListDirty {
		t.Fatalf("did not expect PlayerListDirty to be set")
	}
}
