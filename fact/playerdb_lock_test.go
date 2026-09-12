package fact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ChatWire/cfg"
	"ChatWire/glob"
)

func isolatePlayerChanges(t *testing.T) {
	t.Helper()
	playerChanges.Lock()
	oldNext := playerChanges.next
	oldVersions := playerChanges.versions
	playerChanges.next = 0
	playerChanges.versions = make(map[string]uint64)
	playerChanges.Unlock()
	t.Cleanup(func() {
		playerChanges.Lock()
		playerChanges.next = oldNext
		playerChanges.versions = oldVersions
		playerChanges.Unlock()
	})
}

func TestWritePlayersMergesDirtyRecordsIntoLatestDatabase(t *testing.T) {
	isolatePlayerChanges(t)
	oldGlobal := cfg.Global
	oldLocal := cfg.Local
	oldPlayers := glob.PlayerList
	t.Cleanup(func() {
		cfg.Global = oldGlobal
		cfg.Local = oldLocal
		glob.PlayerList = oldPlayers
	})

	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + string(os.PathSeparator)
	cfg.Global.Paths.DataFiles.DBFile = "playerdb.json"
	cfg.Global.Paths.DataFiles.DBFormat = playerDBFormatJSON
	cfg.Local.Callsign = "server-a"
	dbPath := filepath.Join(root, "playerdb.json")

	remote := map[string]*glob.PlayerData{
		"bob": {Level: 2, Minutes: 50},
	}
	data, err := json.Marshal(remote)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	glob.PlayerList = map[string]*glob.PlayerData{
		"alice": {Name: "alice", Level: 3, Minutes: 100},
		"bob":   {Name: "bob", Level: 2, Minutes: 1}, // stale local copy
	}
	SetPlayerListDirty("alice")
	WritePlayers()

	written, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]*glob.PlayerData
	if err := json.Unmarshal(written, &got); err != nil {
		t.Fatal(err)
	}
	if got["alice"] == nil || got["alice"].Minutes != 100 {
		t.Fatalf("dirty player was not written: %+v", got["alice"])
	}
	if got["bob"] == nil || got["bob"].Minutes != 50 {
		t.Fatalf("unrelated remote update was overwritten: %+v", got["bob"])
	}
	if playerChangePending("alice") {
		t.Fatal("successful save did not clear the dirty player version")
	}

	lockData, err := os.ReadFile(dbPath + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	if len(lockData) == 0 {
		t.Fatal("database lock did not retain diagnostic metadata")
	}
}

func TestMergePlayerChangesSupportsDeletionTombstones(t *testing.T) {
	players := map[string]*glob.PlayerData{
		"alice": {Level: 2},
		"bob":   {Level: 2},
	}
	mergePlayerChanges(players, map[string]*glob.PlayerData{"alice": nil})
	if _, ok := players["alice"]; ok {
		t.Fatal("deletion tombstone did not remove player")
	}
	if players["bob"] == nil {
		t.Fatal("deletion tombstone removed an unrelated player")
	}
}

func TestWritePlayersMigratesLegacyJSONToBinary(t *testing.T) {
	isolatePlayerChanges(t)
	oldGlobal := cfg.Global
	oldLocal := cfg.Local
	oldPlayers := glob.PlayerList
	t.Cleanup(func() {
		cfg.Global = oldGlobal
		cfg.Local = oldLocal
		glob.PlayerList = oldPlayers
	})

	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + string(os.PathSeparator)
	cfg.Global.Paths.DataFiles.DBFile = "legacy-playerdb.json"
	cfg.Global.Paths.DataFiles.DBFormat = playerDBFormatBinary
	cfg.Local.Callsign = "server-a"
	dbPath := filepath.Join(root, "legacy-playerdb.json")

	legacy := map[string]*glob.PlayerData{
		"bob": {Level: 2, Minutes: 50},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	glob.PlayerList = map[string]*glob.PlayerData{
		"alice": {Name: "alice", Level: 3, Minutes: 100},
		"bob":   {Name: "bob", Level: 2, Minutes: 1},
	}
	SetPlayerListDirty("alice")
	WritePlayers()

	written, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	got, format, err := decodePlayerDatabase(written)
	if err != nil {
		t.Fatal(err)
	}
	if format != playerDBFormatBinary {
		t.Fatalf("database format = %q, want %q", format, playerDBFormatBinary)
	}
	if got["alice"] == nil || got["alice"].Minutes != 100 {
		t.Fatalf("dirty player was not migrated: %+v", got["alice"])
	}
	if got["bob"] == nil || got["bob"].Minutes != 50 {
		t.Fatalf("remote player was not preserved: %+v", got["bob"])
	}
}
