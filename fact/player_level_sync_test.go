package fact

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ChatWire/cfg"
	"ChatWire/glob"
)

type playerLevelCapturePipe struct {
	bytes.Buffer
}

func (*playerLevelCapturePipe) Close() error { return nil }

func installPlayerLevelCapturePipe(t *testing.T) *playerLevelCapturePipe {
	t.Helper()
	pipe := &playerLevelCapturePipe{}
	PipeLock.Lock()
	oldPipe := Pipe
	oldGeneration := pipeGeneration
	Pipe = pipe
	pipeGeneration = 0
	PipeLock.Unlock()
	t.Cleanup(func() {
		PipeLock.Lock()
		Pipe = oldPipe
		pipeGeneration = oldGeneration
		PipeLock.Unlock()
	})
	return pipe
}

func decodeSoftModTestRequest(t *testing.T, line string) softModRequest {
	t.Helper()
	line = strings.TrimSpace(line)
	payload, ok := strings.CutPrefix(line, "/chatwire ")
	if !ok {
		t.Fatalf("not a /chatwire request: %q", line)
	}
	decoded, err := DecodeSoftModPayload(payload)
	if err != nil {
		t.Fatalf("invalid request encoding: %v", err)
	}
	var request softModRequest
	if err := json.Unmarshal(decoded, &request); err != nil {
		t.Fatalf("invalid request JSON: %v", err)
	}
	return request
}

func TestApplyPlayerLevelInGameUsesCompleteTransitions(t *testing.T) {
	pipe := installPlayerLevelCapturePipe(t)
	tests := []struct {
		name          string
		previousLevel int
		level         int
		wantWrite     bool
	}{
		{name: "new to member", previousLevel: 0, level: 1, wantWrite: true},
		{name: "member to regular", previousLevel: 1, level: 2, wantWrite: true},
		{name: "regular to veteran", previousLevel: 2, level: 3, wantWrite: true},
		{name: "veteran to moderator", previousLevel: 3, level: 255, wantWrite: true},
		{name: "moderator to veteran", previousLevel: 255, level: 3, wantWrite: true},
		{name: "regular to new", previousLevel: 2, level: 0, wantWrite: true},
		{name: "new player join", previousLevel: 0, level: 0},
		{name: "ban handled separately", previousLevel: 3, level: -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pipe.Reset()
			applyPlayerLevelInGame("alice", tc.previousLevel, tc.level)
			if !tc.wantWrite {
				if got := pipe.String(); got != "" {
					t.Fatalf("commands = %q, want none", got)
				}
				return
			}
			request := decodeSoftModTestRequest(t, pipe.String())
			if request.Command != "player-level" {
				t.Fatalf("command = %q, want player-level", request.Command)
			}
			data := request.Data.(map[string]any)
			if data["name"] != "alice" || int(data["level"].(float64)) != tc.level {
				t.Fatalf("data = %#v, want alice level %d", data, tc.level)
			}
		})
	}
}

func TestPlayerLevelSetFromGameIgnoresAcknowledgement(t *testing.T) {
	isolatePlayerChanges(t)
	oldPlayers := glob.PlayerList
	t.Cleanup(func() { glob.PlayerList = oldPlayers })

	glob.PlayerList = map[string]*glob.PlayerData{
		"alice": {Name: "Alice", Level: 2, LastSeen: 123},
	}
	if !PlayerLevelSetFromGame("Alice", 2) {
		t.Fatal("existing player was not found")
	}
	if playerChangePending("alice") {
		t.Fatal("same-level SoftMod acknowledgement marked the player dirty")
	}
	if got := glob.PlayerList["alice"].LastSeen; got != 123 {
		t.Fatalf("same-level acknowledgement changed LastSeen to %d", got)
	}
}

func TestPlayerLevelSetFromGamePersistsRealPromotion(t *testing.T) {
	isolatePlayerChanges(t)
	oldPlayers := glob.PlayerList
	t.Cleanup(func() { glob.PlayerList = oldPlayers })

	glob.PlayerList = map[string]*glob.PlayerData{
		"alice": {Name: "Alice", Level: 1},
	}
	if !PlayerLevelSetFromGame("Alice", 2) {
		t.Fatal("existing player was not found")
	}
	if got := glob.PlayerList["alice"].Level; got != 2 {
		t.Fatalf("player level = %d, want 2", got)
	}
	if !playerChangePending("alice") {
		t.Fatal("real SoftMod promotion was not marked dirty")
	}
}

func TestSyncOnlinePlayerLevelOnlyWritesForLocalOnlinePlayer(t *testing.T) {
	pipe := installPlayerLevelCapturePipe(t)
	OnlinePlayersLock.Lock()
	oldOnline := glob.OnlinePlayers
	glob.OnlinePlayers = []glob.OnlinePlayerData{{Name: "Alice"}}
	OnlinePlayersLock.Unlock()
	t.Cleanup(func() {
		OnlinePlayersLock.Lock()
		glob.OnlinePlayers = oldOnline
		OnlinePlayersLock.Unlock()
	})

	syncOnlinePlayerLevel("alice", 1, 2)
	syncOnlinePlayerLevel("bob", 1, 2)
	request := decodeSoftModTestRequest(t, pipe.String())
	data := request.Data.(map[string]any)
	if request.Command != "player-level" || data["name"] != "Alice" || data["level"] != float64(2) {
		t.Fatalf("unexpected request: %#v", request)
	}
}

func TestLoadPlayersAppliesRemoteLevelChangeToOnlinePlayer(t *testing.T) {
	isolatePlayerChanges(t)
	pipe := installPlayerLevelCapturePipe(t)

	oldGlobal := cfg.Global
	oldLocal := cfg.Local
	oldPlayers := glob.PlayerList
	OnlinePlayersLock.Lock()
	oldOnline := glob.OnlinePlayers
	OnlinePlayersLock.Unlock()
	t.Cleanup(func() {
		cfg.Global = oldGlobal
		cfg.Local = oldLocal
		glob.PlayerList = oldPlayers
		OnlinePlayersLock.Lock()
		glob.OnlinePlayers = oldOnline
		OnlinePlayersLock.Unlock()
	})

	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + string(os.PathSeparator)
	cfg.Global.Paths.DataFiles.DBFile = "playerdb.json"
	cfg.Global.Paths.DataFiles.DBFormat = playerDBFormatJSON
	cfg.Local = oldLocal
	cfg.Local.Options.MembersOnly = false
	cfg.Local.Options.RegularsOnly = false
	glob.PlayerList = map[string]*glob.PlayerData{
		"alice": {Name: "alice", Level: 1},
	}
	OnlinePlayersLock.Lock()
	glob.OnlinePlayers = []glob.OnlinePlayerData{{Name: "alice", Level: 1}}
	OnlinePlayersLock.Unlock()

	remote := map[string]*glob.PlayerData{
		"alice": {Level: 2},
	}
	data, err := json.Marshal(remote)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "playerdb.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	LoadPlayers(false, false, false)

	request := decodeSoftModTestRequest(t, pipe.String())
	requestData := request.Data.(map[string]any)
	if request.Command != "player-level" || requestData["name"] != "alice" || requestData["level"] != float64(2) {
		t.Fatalf("unexpected cross-server request: %#v", request)
	}
	if got := glob.PlayerList["alice"].Level; got != 2 {
		t.Fatalf("local player level = %d, want 2", got)
	}
}

func TestWhitelistPlayerRemovesIneligiblePlayersAndAdmins(t *testing.T) {
	pipe := installPlayerLevelCapturePipe(t)
	oldLocal := cfg.Local
	oldBooted := FactorioBooted
	oldRunning := FactIsRunning
	t.Cleanup(func() {
		cfg.Local = oldLocal
		FactorioBooted = oldBooted
		FactIsRunning = oldRunning
	})

	FactorioBooted = true
	FactIsRunning = true
	cfg.Local.Options.CustomWhitelist = false
	cfg.Local.Options.MembersOnly = true
	cfg.Local.Options.RegularsOnly = false

	WhitelistPlayer("member", 1)
	WhitelistPlayer("new", 0)
	WhitelistPlayer("admin", 255)
	if got, want := pipe.String(), "/whitelist add member\n/whitelist remove new\n/whitelist remove admin\n"; got != want {
		t.Fatalf("whitelist commands = %q, want %q", got, want)
	}
}
