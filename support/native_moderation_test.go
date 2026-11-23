package support

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
)

type nativeModerationMap struct {
	process          *nativeProcess
	banFile          string
	port, rconOffset int
	password         string
	generation       uint64
	players          map[string]*glob.PlayerData
}

func startNativeModerationMap(t *testing.T, generation uint64) *nativeModerationMap {
	t.Helper()
	n := nativeFixture(t)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(n.binary), "../../data/server-settings.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	settings["name"] = "ChatWire moderation test"
	settings["visibility"] = map[string]bool{"public": false, "lan": false, "steam": false}
	settings["require_user_verification"] = false
	settings["auto_pause"] = false
	settingsPath := filepath.Join(n.root, "server-settings.json")
	writeModerationFixture(t, settingsPath, settings)
	port, rcon := nativeFixedPorts(t)
	m := &nativeModerationMap{
		banFile: filepath.Join(n.root, "server-banlist.json"), port: port, rconOffset: rcon - port,
		password: rand.Text(), generation: generation, players: make(map[string]*glob.PlayerData),
	}
	m.process = n.start(t, "--start-server-load-scenario", "chatwire-native", "--server-settings", settingsPath,
		"--server-banlist", m.banFile, "--bind", "127.0.0.1", "--port", strconv.Itoa(port),
		"--rcon-port", strconv.Itoa(rcon), "--rcon-password", m.password)
	m.process.until(t, isFactorioReadyLine)
	return m
}

func (m *nativeModerationMap) selectMap(t *testing.T) {
	t.Helper()
	glob.PlayerList = m.players
	cfg.Local.Port, cfg.Global.Options.RconOffset, glob.RCONPass = m.port, m.rconOffset, m.password
	fact.SetFactorioPipe(m.process.stdin, m.generation)
	id := fact.WriteSoftModCommand("hello", nil)
	m.until(t, func(e chatWireEnvelope) bool { return e.ID == id && e.OK })
	if !fact.TaggedBanSyncGeneration(m.generation) {
		t.Fatal("Factorio did not negotiate tagged ban sync")
	}
}

func (m *nativeModerationMap) handle(line string) {
	input := preProcessFactorioOutput(line)
	input.generation = m.generation
	if !handleChatWire(input) && !handleBan(input) {
		handleUnBan(input)
	}
}

func (m *nativeModerationMap) until(t *testing.T, match func(chatWireEnvelope) bool) string {
	t.Helper()
	return m.process.until(t, func(line string) bool {
		m.handle(line)
		e, ok := nativeEnvelope(line)
		return ok && match(e)
	})
}

func (m *nativeModerationMap) applied(t *testing.T, name string, revision uint64, banned bool) string {
	t.Helper()
	line := m.until(t, func(e chatWireEnvelope) bool {
		var data struct {
			Name     string `json:"name"`
			Revision uint64 `json:"revision,string"`
		}
		return e.Command == "ban-state" && e.OK && json.Unmarshal(e.Data, &data) == nil && data.Name == name && data.Revision == revision
	})
	m.requireBanFile(t, name, banned)
	return line
}

func (m *nativeModerationMap) requireBanFile(t *testing.T, name string, banned bool) {
	t.Helper()
	data, err := os.ReadFile(m.banFile)
	if err != nil {
		t.Fatal(err)
	}
	var bans []struct {
		Name string `json:"username"`
	}
	if err := json.Unmarshal(data, &bans); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ban := range bans {
		found = found || strings.EqualFold(ban.Name, name)
	}
	if found != banned {
		t.Fatalf("generation %d native ban for %s = %v, want %v", m.generation, name, found, banned)
	}
}

func writeModerationFixture(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNativeFactorioTwoMapLevelRevisions(t *testing.T) {
	a, b := startNativeModerationMap(t, 1001), startNativeModerationMap(t, 1002)
	resetSupportTestState(t)
	oldGlobal, oldLocal, oldPass := cfg.Global, cfg.Local, glob.RCONPass
	t.Cleanup(func() {
		cfg.Global, cfg.Local, glob.RCONPass = oldGlobal, oldLocal, oldPass
		fact.SetFactorioPipe(nil, 0)
	})
	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + string(os.PathSeparator)
	cfg.Global.Paths.DataFiles.DBFile, cfg.Global.Paths.DataFiles.DBFormat = "players.json", "json"
	cfg.Global.Paths.DataFiles.Bans = filepath.Join(root, "community.json")
	cfg.Global.Options.FCLWarnOnly = false
	db := filepath.Join(root, "players.json")
	const player = "nativeunbanprobe"
	const external = "communityrevisionprobe"
	const regular = "communityregularprobe"
	const warning = "communitywarningprobe"
	writeModerationFixture(t, db, map[string]glob.PlayerData{player: {Level: 1}, regular: {Level: 2}, warning: {Level: 1}})
	state := func(name string, level int, revision uint64) glob.PlayerData {
		t.Helper()
		data, err := os.ReadFile(db)
		if err != nil {
			t.Fatal(err)
		}
		var players map[string]glob.PlayerData
		if err := json.Unmarshal(data, &players); err != nil {
			t.Fatal(err)
		}
		got, ok := players[name]
		if !ok || got.Level != level || got.LevelRevision != revision {
			t.Fatalf("%s shared state = %+v, want level %d revision %d", name, got, level, revision)
		}
		return got
	}

	b.selectMap(t)
	fact.LoadPlayers(false, false, false)
	fact.PlayerLevelSetFromGame(player, 2) // Queued before map A's ban.
	a.selectMap(t)
	fact.LoadPlayers(false, false, false)
	if err := fact.SetPlayerLevelByModerator(player, -1, "revision test", "Moderator"); err != nil {
		t.Fatal(err)
	}
	a.applied(t, player, 1, true)
	state(player, -1, 1)

	b.selectMap(t)
	fact.WritePlayers() // The stale promotion must lose the revision comparison.
	fact.LoadPlayers(false, false, false)
	oldReply := b.applied(t, player, 1, true)
	state(player, -1, 1)
	if _, err := fact.ExecuteRCON("unban " + player); err != nil {
		t.Fatal(err)
	}
	b.until(t, func(e chatWireEnvelope) bool { return e.Event == "native-moderation" })
	b.applied(t, player, 2, false)
	b.handle(oldReply)
	b.handle(fmt.Sprintf("2026-11-23 00:00:00 [BAN] %s was banned by <script>. Reason: stale mirror.", player))
	fact.WritePlayers()
	state(player, 0, 2)

	oldRequest := fact.WriteSoftModCommand("ban-state", map[string]any{"name": player, "banned": true, "reason": "old request", "revision": "1"})
	b.until(t, func(e chatWireEnvelope) bool { return e.ID == oldRequest && !e.OK && e.Error == "older ban revision" })
	b.requireBanFile(t, player, false)
	a.selectMap(t)
	fact.LoadPlayers(false, false, false)
	a.applied(t, player, 2, false)

	writeModerationFixture(t, cfg.Global.Paths.DataFiles.Bans, []map[string]string{
		{"username": external, "reason": "peer moderation"}, {"username": regular, "reason": "rank exemption"},
		{"username": warning, "reason": "warn-only check"},
	})
	banlist.ReadBanFile(true)
	cfg.Global.Options.FCLWarnOnly = true
	if banlist.CheckBanList(warning, true) {
		t.Fatal("warn-only community entry was enforced")
	}
	state(warning, 1, 0)
	cfg.Global.Options.FCLWarnOnly = false
	if banlist.CheckBanList(regular, false) || !banlist.CheckBanList(external, true) {
		t.Fatal("community enforcement or Regular exemption failed")
	}
	a.applied(t, external, 1, true)
	if reason := state(external, -1, 1).BanReason; reason != "[FCL] peer moderation" {
		t.Fatalf("community reason = %q", reason)
	}
	state(regular, 2, 0)
	b.selectMap(t)
	fact.LoadPlayers(false, false, false)
	b.applied(t, external, 1, true)

	writeModerationFixture(t, cfg.Global.Paths.DataFiles.Bans, []string{})
	banlist.ReadBanFile(false)
	state(external, -1, 1)
	b.requireBanFile(t, external, true)
	if err := fact.SetPlayerLevelByModerator(external, 0, "", "Moderator"); err != nil {
		t.Fatal(err)
	}
	b.applied(t, external, 2, false)
	a.selectMap(t)
	fact.LoadPlayers(false, false, false)
	a.applied(t, external, 2, false)
	state(external, 0, 2)
}
