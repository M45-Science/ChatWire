package support

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

type testWriteCloser struct {
	bytes.Buffer
}

func (t *testWriteCloser) Close() error { return nil }

func resetSupportTestState(t *testing.T) {
	t.Helper()
	fact.GameMapName = ""
	fact.GameMapPath = ""
	fact.LastSaveName = ""
	fact.ResetGametime(60)
	fact.SetNumPlayers(0)
	fact.FactIsRunning = false
	fact.FactorioBooted = false
	glob.SoftModVersion = constants.Unknown
	glob.OnlineCommand = "/online"
	glob.PlayerList = map[string]*glob.PlayerData{}
	glob.OnlinePlayers = nil
	cfg.Local.Options.HideAutosaves = false
	lastConnectTime = time.Time{}
	lastConnector = ""
	fact.PipeLock.Lock()
	fact.Pipe = nil
	fact.PipeLock.Unlock()
}

func chatWireTestLine(t *testing.T, payload string) *handleData {
	t.Helper()
	encoded, err := fact.EncodeSoftModPayload([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	return preProcessFactorioOutput("[CHATWIRE] " + encoded)
}

func capturedChatWireRequest(t *testing.T, output string) (string, map[string]any) {
	t.Helper()
	start := strings.Index(output, "/chatwire ")
	if start < 0 {
		t.Fatalf("no /chatwire request in %q", output)
	}
	encoded := strings.SplitN(output[start+len("/chatwire "):], "\n", 2)[0]
	payload, err := fact.DecodeSoftModPayload(encoded)
	if err != nil {
		t.Fatalf("invalid protocol encoding: %v", err)
	}
	var request struct {
		Command string         `json:"command"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("invalid protocol JSON: %v", err)
	}
	return request.Command, request.Data
}

func TestHandleMapLoadSetsGameMapState(t *testing.T) {
	resetSupportTestState(t)

	input := preProcessFactorioOutput("0 Loading map /tmp/factorio/saves/test-save.zip: 4096")
	if !handleMapLoad(input) {
		t.Fatal("expected map load line to be handled")
	}
	if fact.GameMapName != "test-save.zip" {
		t.Fatalf("expected game map name to be set, got %q", fact.GameMapName)
	}
	if fact.GameMapPath != "/tmp/factorio/saves/test-save.zip" {
		t.Fatalf("expected game map path to be set, got %q", fact.GameMapPath)
	}
	if fact.LastSaveName != "test-save.zip" {
		t.Fatalf("expected last save name to be updated, got %q", fact.LastSaveName)
	}
}

func TestModLoadStatusDetailIncludesCountAndName(t *testing.T) {
	got := modLoadStatusDetail("Loading mod someModName 1.2.3 (4/24)")
	want := "4/24 someModName"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestHandleSaveMsgUpdatesLastAutosave(t *testing.T) {
	resetSupportTestState(t)

	input := preProcessFactorioOutput("0 Info AppManager.cpp:1: Saving to _autosave7")
	if !handleSaveMsg(input) {
		t.Fatal("expected save message to be handled")
	}
	if fact.LastSaveName != "autosave7" {
		t.Fatalf("expected autosave name to be captured, got %q", fact.LastSaveName)
	}
}

func TestHandleOnlinePlayersUpdatesCount(t *testing.T) {
	resetSupportTestState(t)

	input := preProcessFactorioOutput("Online players (3)")
	if !handleOnlinePlayers(input) {
		t.Fatal("expected online players line to be handled")
	}
	if fact.NumPlayersCurrent() != 3 {
		t.Fatalf("expected 3 players, got %d", fact.NumPlayersCurrent())
	}
}

func TestHandlePlayerLeaveWritesOnlineCommand(t *testing.T) {
	resetSupportTestState(t)
	glob.SoftModVersion = "test"

	w := &testWriteCloser{}
	fact.PipeLock.Lock()
	fact.Pipe = w
	fact.PipeLock.Unlock()
	fact.FactIsRunning = true
	fact.FactorioBooted = true

	input := preProcessFactorioOutput("0 0 [LEAVE] Alice")
	if !handlePlayerLeave(input) {
		t.Fatal("expected leave line to be handled")
	}
	if command, _ := capturedChatWireRequest(t, w.String()); command != "online" {
		t.Fatalf("expected online refresh command, got %q", w.String())
	}
}

func TestHandleBanStoresReason(t *testing.T) {
	resetSupportTestState(t)
	glob.SoftModVersion = "test"

	w := &testWriteCloser{}
	fact.PipeLock.Lock()
	fact.Pipe = w
	fact.PipeLock.Unlock()

	input := preProcessFactorioOutput("0 0 [BAN] Alice was banned by server Reason: griefing")
	if !handleBan(input) {
		t.Fatal("expected ban line to be handled")
	}

	glob.PlayerListLock.RLock()
	player := glob.PlayerList["alice"]
	glob.PlayerListLock.RUnlock()
	if player == nil {
		t.Fatal("expected player to be added")
	}
	if player.Level != -1 {
		t.Fatalf("expected player to be banned, got level %d", player.Level)
	}
	if player.BanReason != "griefing" {
		t.Fatalf("expected ban reason to be stored, got %q", player.BanReason)
	}
	if command, _ := capturedChatWireRequest(t, w.String()); command != "online" {
		t.Fatalf("expected online refresh command, got %q", w.String())
	}
}

func TestHandleBanWithoutReasonColonDoesNotPanic(t *testing.T) {
	resetSupportTestState(t)

	w := &testWriteCloser{}
	fact.PipeLock.Lock()
	fact.Pipe = w
	fact.PipeLock.Unlock()

	input := preProcessFactorioOutput("0 0 [BAN] Bob was banned by server Reason omitted")
	if !handleBan(input) {
		t.Fatal("expected ban line to be handled")
	}

	glob.PlayerListLock.RLock()
	player := glob.PlayerList["bob"]
	glob.PlayerListLock.RUnlock()
	if player == nil {
		t.Fatal("expected player to be added")
	}
	if player.Level != -1 {
		t.Fatalf("expected player to be banned, got level %d", player.Level)
	}
	if player.BanReason != "" {
		t.Fatalf("expected no ban reason to be stored, got %q", player.BanReason)
	}
}

func TestHandleChatWireVeteranPromotionUpdatesPlayerLevel(t *testing.T) {
	resetSupportTestState(t)
	glob.PlayerList["alice"] = &glob.PlayerData{Name: "alice", Level: 2}

	input := chatWireTestLine(t, `{"v":1,"kind":"event","event":"player-level","data":{"name":"Alice","level":3}}`)
	if !handleChatWire(input) {
		t.Fatal("expected veteran promotion message to be handled")
	}
	if got := glob.PlayerList["alice"].Level; got != 3 {
		t.Fatalf("player level = %d, want 3", got)
	}
}

func TestHandleChatWirePromotionAcknowledgementIsNoOp(t *testing.T) {
	resetSupportTestState(t)
	glob.PlayerList["alice"] = &glob.PlayerData{Name: "Alice", Level: 3, LastSeen: 123}

	input := chatWireTestLine(t, `{"v":1,"kind":"event","event":"player-level","data":{"name":"Alice","level":3}}`)
	if !handleChatWire(input) {
		t.Fatal("expected veteran acknowledgement to be handled")
	}
	player := glob.PlayerList["alice"]
	if player.Level != 3 || player.LastSeen != 123 {
		t.Fatalf("acknowledgement mutated player: %+v", player)
	}
}

func TestHandleIncomingAnnouncePausesForConnect(t *testing.T) {
	resetSupportTestState(t)

	w := &testWriteCloser{}
	fact.PipeLock.Lock()
	fact.Pipe = w
	fact.PipeLock.Unlock()
	glob.PausedForConnect = true
	glob.PausedFor = "Bob"
	defer func() {
		glob.PausedForConnect = false
		glob.PausedFor = ""
		glob.PausedConnectAttempt = false
	}()

	input := preProcessFactorioOutput("0 Queuing ban recommendation check for user Bob")
	if !handleIncomingAnnounce(input) {
		t.Fatal("expected incoming announce line to be handled")
	}
	if !glob.PausedConnectAttempt {
		t.Fatal("expected paused connect attempt flag to be set")
	}
	if command, data := capturedChatWireRequest(t, w.String()); command != "config" || math.Abs(data["speed"].(float64)-(4.0/60.0)) > 0.000001 {
		t.Fatalf("expected pause-speed request, got command=%q data=%#v", command, data)
	}
}

func TestHandleIncomingAnnounceDeduplicatesRecentConnector(t *testing.T) {
	resetSupportTestState(t)

	lastConnectTime = time.Now()
	lastConnector = "Eve"

	input := preProcessFactorioOutput("0 Queuing ban recommendation check for user Eve")
	if !handleIncomingAnnounce(input) {
		t.Fatal("expected incoming announce line to be handled")
	}
	if lastConnector != "Eve" {
		t.Fatalf("expected last connector to remain Eve, got %q", lastConnector)
	}
	if time.Since(lastConnectTime) > time.Second {
		t.Fatal("expected last connect time to be refreshed")
	}
}

func TestIsFactorioReadyLineMatchesLessStrictRconStartup(t *testing.T) {
	cases := []string{
		"Info RemoteCommandProcessor.cpp:123: Starting RCON interface at IP ADDR:({127.0.0.1:27015})",
		"Starting RCON interface at IP ADDR:({127.0.0.1:27015})",
		"0.552 2026-04-21 13:05:00 Starting RCON interface at IP ADDR:({0.0.0.0:27015})",
	}

	for _, tc := range cases {
		if !isFactorioReadyLine(tc) {
			t.Fatalf("expected ready line to match: %q", tc)
		}
	}

	if isFactorioReadyLine("Info AppManagerStates.cpp:2111: Saving finished") {
		t.Fatal("did not expect unrelated line to match")
	}
}

func TestHandleFactReadyMatchesWithoutLegacyPrefix(t *testing.T) {
	resetSupportTestState(t)

	w := &testWriteCloser{}
	fact.PipeLock.Lock()
	fact.Pipe = w
	fact.PipeLock.Unlock()

	input := preProcessFactorioOutput("0 Starting RCON interface at IP ADDR:({127.0.0.1:27015})")
	handleFactReady(input)

	got := w.String()
	command, _ := capturedChatWireRequest(t, got)
	if command != "hello" {
		t.Fatalf("expected ready handling to request version, got %q", got)
	}
	if !strings.Contains(got, "\n/time\n") {
		t.Fatalf("expected ready handling to initialize native game time, got %q", got)
	}
}

func TestChatWireLinesAreCritical(t *testing.T) {
	if !isCriticalFactorioLine(`[CHATWIRE] {"v":1,"kind":"event","event":"online","data":{}}`) {
		t.Fatal("protocol messages must bypass a full stdout queue")
	}
}

func TestHandleChatWireOnlineUsesExactTicks(t *testing.T) {
	resetSupportTestState(t)
	fact.SetNumPlayers(3)
	input := chatWireTestLine(t, `{"v":1,"kind":"event","event":"online","data":{"count":1,"players":[{"name":"Alice","score_ticks":12345,"time_ticks":67890,"type":"Regulars","afk":"3m"}]}}`)
	if !handleChatWire(input) {
		t.Fatal("expected protocol line to be handled")
	}
	if got := fact.NumPlayersCurrent(); got != 1 {
		t.Fatalf("online count = %d, want 1", got)
	}
	fact.OnlinePlayersLock.RLock()
	defer fact.OnlinePlayersLock.RUnlock()
	if len(glob.OnlinePlayers) != 1 || glob.OnlinePlayers[0].ScoreTicks != 12345 || glob.OnlinePlayers[0].TimeTicks != 67890 || glob.OnlinePlayers[0].Level != 2 {
		t.Fatalf("online players = %#v", glob.OnlinePlayers)
	}
}

func TestHandleChatWireRejectsMalformedJSON(t *testing.T) {
	resetSupportTestState(t)
	if !handleChatWire(preProcessFactorioOutput(`[CHATWIRE] nope`)) {
		t.Fatal("malformed protocol line should still be consumed")
	}
}
