package fact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	playerChanges.versions = make(map[string]playerChangeVersions)
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
	if playerRecordChangePending("alice") {
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

func TestStatsSavePreservesRemoteBanAndDoesNotResurrectDeletedPlayer(t *testing.T) {
	configurePlayerDBTest(t, `{"alice":{"l":-1,"b":"griefing"},"carol":{"l":2,"m":50}}`)
	glob.PlayerList = map[string]*glob.PlayerData{
		"alice": {Name: "alice", Level: 1, LastSeen: 120, Minutes: 100},
		"bob":   {Name: "bob", Level: 1, LastSeen: 120, Minutes: 100},
		"carol": {Name: "carol", Level: 1, LastSeen: 120, Minutes: 100},
	}
	for name := range glob.PlayerList {
		SetPlayerStatsDirty(name)
	}
	WritePlayers()

	data, err := os.ReadFile(cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile)
	if err != nil {
		t.Fatal(err)
	}
	players, _, err := decodePlayerDatabase(data)
	if err != nil {
		t.Fatal(err)
	}
	if alice := players["alice"]; alice == nil || alice.Level != -1 || alice.BanReason != "griefing" || alice.Minutes != 0 {
		t.Fatalf("stats save changed the banned record: %+v", alice)
	}
	if bob := players["bob"]; bob != nil {
		t.Fatalf("stats save restored the deleted record: %+v", bob)
	}
	if carol := players["carol"]; carol == nil || carol.Level != 2 || carol.LastSeen != 120 || carol.Minutes != 100 {
		t.Fatalf("stats save failed to merge into the latest record: %+v", carol)
	}

	pipe := installPlayerLevelCapturePipe(t)
	LoadPlayers(false, false, false)
	if alice := glob.PlayerList["alice"]; alice.Level != -1 || !strings.Contains(pipe.String(), "/ban alice griefing\n") {
		t.Fatalf("reload failed to apply the remote ban: player=%+v commands=%q", alice, pipe.String())
	}
}

func TestBootReloadPreservesPendingBan(t *testing.T) {
	configurePlayerDBTest(t, `{"alice":{"l":1}}`)
	glob.PlayerList = map[string]*glob.PlayerData{"alice": {Name: "alice", Level: 1}}
	pipe := installPlayerLevelCapturePipe(t)
	PlayerSetBanReason("alice", "griefing", false)

	LoadPlayers(true, false, false)

	if alice := glob.PlayerList["alice"]; alice.Level != -1 || alice.BanReason != "griefing" || pipe.Len() != 0 {
		t.Fatalf("boot reload reversed the pending ban: player=%+v commands=%q", alice, pipe.String())
	}
}

func TestReloadAppliesRemoteUnbanDuringPendingStatsChange(t *testing.T) {
	configurePlayerDBTest(t, `{"alice":{"l":0}}`)
	glob.PlayerList = map[string]*glob.PlayerData{"alice": {Name: "alice", Level: -1}}
	pipe := installPlayerLevelCapturePipe(t)
	SetPlayerStatsDirty("alice")

	LoadPlayers(false, false, false)

	if alice := glob.PlayerList["alice"]; alice.Level != 0 || !strings.Contains(pipe.String(), "/unban alice\n") {
		t.Fatalf("reload missed a stored unban: player=%+v commands=%q", alice, pipe.String())
	}
}

func TestStatsUpdateDuringSaveDoesNotReplaySavedRecord(t *testing.T) {
	configurePlayerDBTest(t, `{"alice":{"l":-1,"b":"griefing"}}`)
	glob.PlayerList = map[string]*glob.PlayerData{"alice": {Name: "alice", Level: 1}}
	SetPlayerListDirty("alice")
	saved := snapshotPlayerChanges()
	SetPlayerStatsDirty("alice")
	completePlayerChanges(saved)

	pending := snapshotPlayerChanges()["alice"].versions
	if pending.record != 0 || pending.stats == 0 {
		t.Fatalf("pending changes after save = %+v, want stats only", pending)
	}
}

func configurePlayerDBTest(t *testing.T, contents string) {
	t.Helper()
	isolatePlayerChanges(t)
	oldGlobal, oldPlayers := cfg.Global, glob.PlayerList
	t.Cleanup(func() {
		cfg.Global, glob.PlayerList = oldGlobal, oldPlayers
	})
	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + string(os.PathSeparator)
	cfg.Global.Paths.DataFiles.DBFile = "playerdb.json"
	cfg.Global.Paths.DataFiles.DBFormat = playerDBFormatJSON
	if err := os.WriteFile(filepath.Join(root, "playerdb.json"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestMergePlayerChangesSupportsDeletionTombstones(t *testing.T) {
	players := map[string]*glob.PlayerData{
		"alice": {Level: 2},
		"bob":   {Level: 2},
	}
	mergePlayerChanges(players, map[string]playerChange{"alice": {versions: playerChangeVersions{record: 1}}})
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
