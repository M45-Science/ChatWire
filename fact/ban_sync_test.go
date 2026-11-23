package fact

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/glob"
)

func TestTaggedBanRetriesCurrentRevisionAfterLostReply(t *testing.T) {
	pipe := installPlayerLevelCapturePipe(t)
	banSync.Lock()
	oldPlayers, oldEnabled, oldGeneration := banSync.players, banSync.enabled, banSync.generation
	banSync.players, banSync.enabled = make(map[string]banSyncState), false
	banSync.Unlock()
	t.Cleanup(func() {
		banSync.Lock()
		banSync.players, banSync.enabled, banSync.generation = oldPlayers, oldEnabled, oldGeneration
		banSync.Unlock()
	})
	EnableTaggedBanSync(0)
	rememberBanState("alice", &glob.PlayerData{Level: -1, BanRevision: 5}, false)
	ReconcilePlayerBans()
	old := decodeSoftModTestRequest(t, pipe.String())
	pipe.Reset()
	rememberBanState("alice", &glob.PlayerData{Level: 0, BanRevision: 6}, false)
	ReconcilePlayerBans()
	first := decodeSoftModTestRequest(t, pipe.String())
	ConfirmTaggedBan(0, old.ID, "alice", "5", true)

	banSync.Lock()
	state := banSync.players["alice"]
	state.sent = time.Now().Add(-time.Minute)
	banSync.players["alice"] = state
	banSync.Unlock()
	pipe.Reset()
	ReconcilePlayerBans()
	retry := decodeSoftModTestRequest(t, pipe.String())
	data := retry.Data.(map[string]any)
	if retry.ID == first.ID || data["revision"] != "6" || data["banned"] != false {
		t.Fatalf("retry = %+v, want a fresh request for revision 6 unban", retry)
	}
	ConfirmTaggedBan(0, first.ID, "alice", "6", false)
	ConfirmTaggedBan(0, retry.ID, "alice", "6", false)
	banSync.Lock()
	defer banSync.Unlock()
	if !banSync.players["alice"].applied {
		t.Fatal("latest reply did not confirm the unban")
	}
}

func TestNativeModerationRetryPreservesRevision(t *testing.T) {
	for _, newerBan := range []bool{false, true} {
		name := "retrycurrent"
		if newerBan {
			name = "retryconflict"
		}
		t.Run(name, func(t *testing.T) {
			configurePlayerDBTest(t, `invalid temporary data`)
			glob.PlayerList = map[string]*glob.PlayerData{}
			installPlayerLevelCapturePipe(t)
			if err := RecordNativeModeration(name, false, "", 2); err == nil {
				t.Fatal("native unban unexpectedly saved to invalid database")
			}
			player := &glob.PlayerData{Level: -1, LevelRevision: 2, BanRevision: 2, BanReason: "ban"}
			if newerBan {
				player.LevelRevision, player.BanRevision = 4, 4
			}
			writeRevisionFixture(t, map[string]*glob.PlayerData{name: player})
			RetryNativeModeration()
			got := readRevisionFixture(t)[name]
			if newerBan && (!got.Banned() || got.LevelRevision != 4) {
				t.Fatalf("stale native retry changed the newer ban: %+v", got)
			}
			if !newerBan && (got.Banned() || got.LevelRevision != 3) {
				t.Fatalf("retry did not commit the native unban: %+v", got)
			}
		})
	}
}

func TestRegistrationAndModeratorActionRespectCommittedRevision(t *testing.T) {
	configurePlayerDBTest(t, `{"alice":{"l":-1,"lr":2,"br":2,"b":"newer ban"}}`)
	installPlayerLevelCapturePipe(t)
	stale := func() {
		glob.PlayerList = map[string]*glob.PlayerData{"alice": {Name: "alice", Level: 1, LevelRevision: 1}}
	}
	stale()
	if err := SetPlayerLevelByModerator("alice", 0, "", "Moderator"); !errors.Is(err, errPlayerLevelConflict) {
		t.Fatalf("moderator action = %v, want a revision conflict", err)
	}
	stale()
	if err := RegisterPlayer("alice", "123", 2); !errors.Is(err, errPlayerLevelConflict) {
		t.Fatalf("registration = %v, want a revision conflict", err)
	}
	if err := RegisterPlayer("alice", "123", 2); err == nil {
		t.Fatal("fresh registration unexpectedly changed a ban")
	}
	if got := readRevisionFixture(t)["alice"]; !got.Banned() || got.LevelRevision != 2 || got.ID != "" {
		t.Fatalf("stored moderator decision changed: %+v", got)
	}
	if err := SetPlayerLevelByModerator("alice", 0, "", "Moderator"); err != nil {
		t.Fatal(err)
	}
	if got := readRevisionFixture(t)["alice"]; got.Banned() || got.LevelRevision != 3 {
		t.Fatalf("manual unban did not advance the revision: %+v", got)
	}
}

func TestDeletingProfileRetainsItsNativeBan(t *testing.T) {
	configurePlayerDBTest(t, `{"alice":{"l":-1,"lr":2,"br":2,"b":"ban"}}`)
	glob.PlayerList = make(map[string]*glob.PlayerData)
	installPlayerLevelCapturePipe(t)
	LoadPlayers(true, false, false)
	if err := SetPlayerLevelByModerator("alice", -255, "", "Moderator"); err != nil {
		t.Fatal(err)
	}
	if got := readRevisionFixture(t)["alice"]; got.Level != -255 || !got.Banned() || got.LevelRevision != 3 || got.BanRevision != 2 {
		t.Fatalf("profile deletion changed native ban state: %+v", got)
	}
	if err := RecordNativeModeration("alice", false, "", 2); err != nil {
		t.Fatal(err)
	}
	if got := readRevisionFixture(t)["alice"]; got.Banned() || got.LevelRevision != 4 || got.BanRevision != 4 {
		t.Fatalf("explicit native unban did not clear the retained ban: %+v", got)
	}
}

func writeRevisionFixture(t *testing.T, players map[string]*glob.PlayerData) {
	t.Helper()
	data, err := encodePlayerDatabase(players, playerDBFormatJSON)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.Global.Paths.Folders.ServersRoot, cfg.Global.Paths.DataFiles.DBFile)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func readRevisionFixture(t *testing.T) map[string]*glob.PlayerData {
	t.Helper()
	path := filepath.Join(cfg.Global.Paths.Folders.ServersRoot, cfg.Global.Paths.DataFiles.DBFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	players, _, err := decodePlayerDatabase(data)
	if err != nil {
		t.Fatal(err)
	}
	return players
}
