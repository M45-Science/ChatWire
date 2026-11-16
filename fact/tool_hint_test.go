package fact

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/glob"
)

func TestToolHintCooldownPersistsAcrossInstancesAndStaleWrites(t *testing.T) {
	isolatePlayerChanges(t)
	oldGlobal, oldLocal, oldPlayers := cfg.Global, cfg.Local, glob.PlayerList
	t.Cleanup(func() { cfg.Global, cfg.Local, glob.PlayerList = oldGlobal, oldLocal, oldPlayers })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir() + string(os.PathSeparator)
	cfg.Global.Paths.DataFiles.DBFile = "players.json"
	cfg.Local.Callsign = "hint-test"
	now := time.Unix(1800000000, 0)
	for _, format := range []string{playerDBFormatJSON, playerDBFormatBinary} {
		cfg.Global.Paths.DataFiles.DBFormat = format
		for _, level := range []int{0, 1, 2, 3, 255} {
			glob.PlayerList = map[string]*glob.PlayerData{"alice": {Name: "Alice", Level: level}, "bob": {Level: 2}}
			path := filepath.Join(cfg.Global.Paths.Folders.ServersRoot, cfg.Global.Paths.DataFiles.DBFile)
			initial, err := encodePlayerDatabase(glob.PlayerList, format)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, initial, 0600); err != nil {
				t.Fatal(err)
			}
			if allowed, err := ClaimToolHint("ALICE", now); err != nil || !allowed {
				t.Fatalf("first claim: %v %v", allowed, err)
			}
			// Another instance still has a pre-claim in-memory record.
			glob.PlayerList["alice"].LastToolHint = 0
			if allowed, err := ClaimToolHint("Alice", now.Add(time.Hour)); err != nil || allowed {
				t.Fatalf("reconnect bypassed cooldown: %v %v", allowed, err)
			}
			SetPlayerListDirty("alice")
			WritePlayers()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			players, _, err := decodePlayerDatabase(data)
			if err != nil || players["alice"].LastToolHint != now.Unix() || players["bob"].Level != 2 {
				t.Fatalf("stale write lost timestamp or other player: %v %v", players, err)
			}
			LoadPlayers(false, false, false)
			if glob.PlayerList["alice"].LastToolHint != now.Unix() {
				t.Fatal("database load lost timestamp")
			}
			interval := 24 * time.Hour
			if level > 1 {
				interval = 30 * 24 * time.Hour
			}
			if allowed, err := ClaimToolHint("Alice", now.Add(interval-time.Second)); err != nil || allowed {
				t.Fatalf("early claim: %v %v", allowed, err)
			}
			if allowed, err := ClaimToolHint("Alice", now.Add(interval)); err != nil || !allowed {
				t.Fatalf("due claim level %d: %v %v", level, allowed, err)
			}
			if allowed, err := ClaimToolHint("Alice", now); err != nil || allowed {
				t.Fatalf("clock reversal bypassed cooldown: %v %v", allowed, err)
			}
		}
	}
}

func TestToolHintFailuresDoNotConsumeCooldown(t *testing.T) {
	oldGlobal, oldPlayers := cfg.Global, glob.PlayerList
	t.Cleanup(func() { cfg.Global, glob.PlayerList = oldGlobal, oldPlayers })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir() + string(os.PathSeparator)
	cfg.Global.Paths.DataFiles.DBFile = "players.json"
	glob.PlayerList = map[string]*glob.PlayerData{"alice": {Level: 1}, "banned": {Level: -1}}
	for _, name := range []string{"missing", "banned", ""} {
		if allowed, err := ClaimToolHint(name, time.Now()); err != nil || allowed {
			t.Fatalf("unexpected grant for %q: %v %v", name, allowed, err)
		}
	}
	path := filepath.Join(cfg.Global.Paths.Folders.ServersRoot, cfg.Global.Paths.DataFiles.DBFile)
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if allowed, err := ClaimToolHint("Alice", time.Now()); err == nil || allowed {
		t.Fatalf("corrupt database: %v %v", allowed, err)
	}
	if glob.PlayerList["alice"].LastToolHint != 0 {
		t.Fatal("failure consumed cooldown")
	}
}

func TestToolHintBinaryV1Migration(t *testing.T) {
	// Construct a real legacy v1 record rather than using the current encoder.
	payload := binary.AppendUvarint(nil, 1)
	payload, _ = appendPlayerDBString(payload, "alice")
	payload = binary.AppendVarint(payload, 1)
	payload, _ = appendPlayerDBString(payload, "")
	payload, _ = appendPlayerDBString(payload, "")
	for _, value := range []int64{10, 20, 30, 40} {
		payload = binary.AppendVarint(payload, value)
	}
	if err := initPlayerDBCodecs(); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, playerDBHeaderSize)
	copy(data, playerDBMagic[:])
	binary.LittleEndian.PutUint16(data[8:10], 1)
	data[10] = playerDBCodecZstd
	binary.LittleEndian.PutUint64(data[12:20], uint64(len(payload)))
	binary.LittleEndian.PutUint32(data[20:24], crc32.Checksum(payload, playerDBCRC))
	data = append(data, playerDBEncoder.EncodeAll(payload, nil)...)
	players, _, err := decodePlayerDatabase(data)
	if err != nil || players["alice"].LastToolHint != 0 || players["alice"].Minutes != 30 {
		t.Fatalf("v1 migration: %v %v", players, err)
	}
	players["alice"].LastToolHint = 1800000000
	updated, err := encodePlayerDatabase(players, playerDBFormatBinary)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := decodePlayerDatabase(updated)
	if err != nil || decoded["alice"].LastToolHint != 1800000000 {
		t.Fatalf("v2 timestamp round trip: %v %v", decoded, err)
	}
}
