package support

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
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
	fact.Gametime = "12:34"
	fact.SetNumPlayers(0)
	fact.FactIsRunning = false
	fact.FactorioBooted = false
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
	if got := w.String(); got != "/online\n" {
		t.Fatalf("expected online refresh command, got %q", got)
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
	if got := w.String(); !strings.Contains(got, "/aspeed 4\n") {
		t.Fatalf("expected pause-speed command in output, got %q", got)
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
	if !strings.Contains(got, "/sversion\n") {
		t.Fatalf("expected ready handling to request version, got %q", got)
	}
	if !strings.Contains(got, "/online\n") {
		t.Fatalf("expected ready handling to request player list, got %q", got)
	}
}
