package support

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

// Opt-in: ordinary go test never downloads Factorio or touches an installation.
// Every engine process uses a private config, write-data path, mods and saves.
type nativeFactorio struct {
	binary, root, config, mods string
}

func nativeFixture(t *testing.T) nativeFactorio {
	t.Helper()
	binary := os.Getenv("CW_FACTORIO_BINARY")
	if binary == "" {
		t.Skip("set CW_FACTORIO_BINARY to a Factorio 2.1.24+ headless executable")
	}
	var err error
	binary, err = filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	version, err := exec.CommandContext(ctx, binary, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("Factorio --version: %v\n%s", err, version)
	}
	t.Log(strings.SplitN(string(version), "\n", 2)[0])
	help, err := exec.CommandContext(ctx, binary, "--help").CombinedOutput()
	if err != nil || !strings.Contains(string(help), "--test-scenario") {
		t.Fatal("Factorio executable must support native --test-scenario/--test-save")
	}
	softmod := os.Getenv("CW_SOFTMOD_SOURCE")
	if softmod == "" {
		softmod = "../../SoftMod"
	}
	softmod, err = filepath.Abs(softmod)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	n := nativeFactorio{binary: binary, root: root, config: filepath.Join(root, "config.ini"), mods: filepath.Join(root, "mods")}
	scenario := filepath.Join(root, "scenarios", "chatwire-native")
	for _, dir := range []string{scenario, n.mods} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Copy runtime assets only; never copy the repository's .git or local config.
	err = filepath.WalkDir(softmod, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(softmod, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if rel != "." && rel != "img" && rel != "locale" && !strings.HasPrefix(rel, "img"+string(filepath.Separator)) && !strings.HasPrefix(rel, "locale"+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(scenario, rel), 0755)
		}
		if !strings.HasSuffix(rel, ".lua") && !strings.HasPrefix(rel, "img"+string(filepath.Separator)) && !strings.HasPrefix(rel, "locale"+string(filepath.Separator)) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(scenario, rel), data, 0600)
	})
	if err != nil {
		t.Fatalf("stage SoftMod (%s): %v", softmod, err)
	}
	control, err := os.ReadFile(filepath.Join(scenario, "control.lua"))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := os.ReadFile("testdata/native-factorio.lua")
	if err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(scenario, "control.lua"), append(append(control, '\n'), runner...))
	data := filepath.Clean(filepath.Join(filepath.Dir(binary), "../../data"))
	write(n.config, []byte(fmt.Sprintf("[path]\nread-data=%s\nwrite-data=%s\n[general]\nlocale=en\n", data, root)))
	write(filepath.Join(root, "player-data.json"), []byte(`{"service-username":"NativeTester"}`))
	write(filepath.Join(n.mods, "mod-list.json"), []byte(`{"mods":[{"name":"base","enabled":true},{"name":"quality","enabled":true},{"name":"space-age","enabled":false},{"name":"elevated-rails","enabled":false}]}`))
	return n
}

func (n nativeFactorio) command(ctx context.Context, args ...string) *exec.Cmd {
	base := []string{"--config", n.config, "--mod-directory", n.mods}
	return exec.CommandContext(ctx, n.binary, append(base, args...)...)
}

func (n nativeFactorio) run(t *testing.T, save string, arg any) string {
	t.Helper()
	data, err := json.Marshal(arg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	args := []string{"--test-scenario", "chatwire-native"}
	if save != "" {
		args = []string{"--test-save", save}
	}
	args = append(args, "--test-ticks", "1200", "--test-arg", string(data))
	out, err := n.command(ctx, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("native Factorio: %v\n%s", err, out)
	}
	// --test-ticks returns success on timeout too. Require our explicit assertion marker.
	if !strings.Contains(string(out), "NATIVE-FACTORIO-PASS ") {
		t.Fatalf("native assertions never completed:\n%s", out)
	}
	return string(out)
}

func nativeEnvelope(line string) (chatWireEnvelope, bool) {
	var envelope chatWireEnvelope
	if !strings.HasPrefix(line, chatWireTag) {
		return envelope, false
	}
	data, err := fact.DecodeSoftModPayload(strings.TrimSpace(strings.TrimPrefix(line, chatWireTag)))
	if err != nil || json.Unmarshal(data, &envelope) != nil {
		return envelope, false
	}
	return envelope, envelope.Version == fact.SoftModProtocolVersion
}

func TestNativeFactorioQuickbarPersistence(t *testing.T) {
	n := nativeFixture(t)
	resetSupportTestState(t)
	oldGlobal, oldLocal := cfg.Global, cfg.Local
	t.Cleanup(func() { cfg.Global, cfg.Local = oldGlobal, oldLocal; fact.SetFactorioPipe(nil, 0) })
	cfg.Global.Paths.Folders.ServersRoot = n.root
	output := n.run(t, "", map[string]any{"mode": "export"})
	var request quickbarRequest
	for _, line := range strings.Split(output, "\n") {
		if e, ok := nativeEnvelope(line); ok && e.Event == "quickbar-preset" {
			var candidate quickbarRequest
			if err := json.Unmarshal(e.Data, &candidate); err != nil {
				t.Fatal(err)
			}
			if candidate.Action == "store" {
				request = candidate
			}
		}
	}
	if request.Exchange == "" || request.Slot != 2 {
		t.Fatalf("no real GUI store request:\n%s", output)
	}
	glob.PlayerList[strings.ToLower(request.Name)] = &glob.PlayerData{Name: request.Name, Level: 1}
	w := &testWriteCloser{}
	fact.SetFactorioPipe(w, 0)
	data, _ := json.Marshal(request)
	handleQuickbarPreset(data)
	_, pending := capturedChatWireRequest(t, w.String())
	if pending["pending"] != true {
		t.Fatal("store confirmed before persistence")
	}
	path := filepath.Join(n.root, "quickbar-presets.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("store bypassed the batch")
	}
	w.Reset()
	FlushQuickbarPresets()
	_, saved := capturedChatWireRequest(t, w.String())
	if saved["error"] != nil || saved["pending"] != nil {
		t.Fatalf("save failed: %v", saved)
	}
	// Read from the committed file using a fresh request, not an in-memory exchange.
	w.Reset()
	request.Action, request.ID, request.Exchange = "load", request.ID+1, ""
	data, _ = json.Marshal(request)
	handleQuickbarPreset(data)
	_, reply := capturedChatWireRequest(t, w.String())
	if reply["exchange"] == nil {
		t.Fatalf("committed preset missing: %v", reply)
	}
	n.run(t, "", map[string]any{"mode": "import", "reply": reply})
}

func TestNativeFactorioDepletedMinerModeration(t *testing.T) {
	n := nativeFixture(t)
	resetSupportTestState(t)
	oldMute := cfg.Global.Options.ShutupSusWarn
	cfg.Global.Options.ShutupSusWarn = false
	clear(mapActivityAlerts.sites)
	clear(mapActivityAlerts.sessions)
	t.Cleanup(func() {
		cfg.Global.Options.ShutupSusWarn = oldMute
		clear(mapActivityAlerts.sites)
		clear(mapActivityAlerts.sessions)
	})
	out := n.run(t, "", map[string]any{"mode": "deconstruction"})
	if strings.Contains(out, "NATIVE-DEPLETED-PLANNER-PASS") {
		t.Log("verified the actual depleted-mining-drill planner filter")
	}
	phase, player := "", ""
	counts := map[string]int{}
	cancellations := 0
	check := func() {
		t.Helper()
		if phase == "depleted" {
			if counts[phase] != 64 {
				t.Fatalf("expected 64 auditable depleted marks, got %d", counts[phase])
			}
			if p := glob.PlayerList[strings.ToLower(player)]; p.SusScore != 0 {
				t.Errorf("depleted cleanup added suspicion: %d", p.SusScore)
			}
			if len(mapActivityAlerts.due(time.Now().Add(time.Minute))) != 0 {
				t.Error("legitimate depleted cleanup triggered moderation")
			}
		}
	}
	for _, line := range strings.Split(out, "\n") {
		e, ok := nativeEnvelope(line)
		if !ok {
			continue
		}
		if e.Event == "native-phase" {
			check()
			var data struct{ Name, Player string }
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatal(err)
			}
			phase, player = data.Name, data.Player
			if glob.PlayerList[strings.ToLower(player)] == nil {
				glob.PlayerList[strings.ToLower(player)] = &glob.PlayerData{Name: player}
			}
		} else if e.Event == "activity" {
			var data struct{ Text string }
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(data.Text, " decon ") || strings.Contains(data.Text, " decon-depleted ") {
				counts[phase]++
			}
			if strings.Contains(data.Text, " undecon-depleted ") {
				cancellations++
			}
			handleChatWireEvent(e.Event, e.Data)
		}
	}
	check()
	if counts["live"] != 1 || counts["structures"] != 64 {
		t.Fatalf("missing live-drill/structure audit events: %v", counts)
	}
	if cancellations != 64 {
		t.Fatalf("missing depleted cancellation audit events: %d", cancellations)
	}
	alerts := mapActivityAlerts.due(time.Now().Add(time.Minute))
	if len(alerts) != 1 || !strings.Contains(alerts[0].message, "65 marks") || strings.Contains(alerts[0].message, "cancellation") {
		t.Fatalf("live drill/structures lost moderation: %+v", alerts)
	}
}

type nativeProcess struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	lines chan string
	done  chan struct{}
	mu    sync.Mutex
	log   strings.Builder
	err   error
}

func (p *nativeProcess) output() string { p.mu.Lock(); defer p.mu.Unlock(); return p.log.String() }

func (n nativeFactorio) start(t *testing.T, args ...string) *nativeProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	p := &nativeProcess{cmd: n.command(ctx, args...), lines: make(chan string, 4096), done: make(chan struct{})}
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	p.cmd.Stderr = p.cmd.Stdout
	if err := p.cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 65536), 1<<20)
		for scanner.Scan() {
			line := scanner.Text()
			p.mu.Lock()
			p.log.WriteString(line + "\n")
			p.mu.Unlock()
			select {
			case p.lines <- line:
			case <-ctx.Done():
			}
		}
		p.err = p.cmd.Wait()
		close(p.lines)
		close(p.done)
	}()
	t.Cleanup(func() { cancel(); <-p.done; p.stdin.Close() })
	return p
}

func (p *nativeProcess) until(t *testing.T, match func(string) bool) string {
	t.Helper()
	timer := time.NewTimer(40 * time.Second)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				t.Fatalf("Factorio exited before expected output: %v\n%s", p.err, p.output())
			}
			if match(line) {
				return line
			}
		case <-timer.C:
			t.Fatalf("Factorio output timeout:\n%s", p.output())
		}
	}
}

func nativeFixedPorts(t *testing.T) (int, int) {
	t.Helper()
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer tcp.Close()
	return udp.LocalAddr().(*net.UDPAddr).Port, tcp.Addr().(*net.TCPAddr).Port
}

func TestNativeFactorioFixedPortLifecycle(t *testing.T) {
	for _, nonblocking := range []bool{false, true} {
		t.Run(fmt.Sprintf("nonblocking=%v", nonblocking), func(t *testing.T) { nativeFixedPortLifecycle(t, nonblocking) })
	}
}

func nativeFixedPortLifecycle(t *testing.T, nonblocking bool) {
	t.Helper()
	n := nativeFixture(t)
	resetSupportTestState(t)
	oldGlobal, oldLocal, oldPass := cfg.Global, cfg.Local, glob.RCONPass
	t.Cleanup(func() {
		cfg.Global, cfg.Local, glob.RCONPass = oldGlobal, oldLocal, oldPass
		fact.SetFactorioPipe(nil, 0)
	})
	port, rcon := nativeFixedPorts(t)
	cfg.Local.Port, cfg.Global.Options.RconOffset = port, rcon-port
	glob.RCONPass = rand.Text()
	settings := filepath.Join(n.root, "server-settings.json")
	example, err := os.ReadFile(filepath.Clean(filepath.Join(filepath.Dir(n.binary), "../../data/server-settings.example.json")))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(example, &config); err != nil {
		t.Fatal(err)
	}
	config["name"], config["visibility"] = "Chatwire native test", map[string]bool{"public": false, "lan": false, "steam": false}
	config["require_user_verification"], config["auto_pause"], config["non_blocking_saving"] = false, false, nonblocking
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	serverArgs := []string{"--server-settings", settings, "--bind", "127.0.0.1", "--port", fmt.Sprint(port), "--rcon-port", fmt.Sprint(rcon), "--rcon-password", glob.RCONPass}
	server := n.start(t, append([]string{"--start-server-load-scenario", "chatwire-native"}, serverArgs...)...)
	ready := server.until(t, isFactorioReadyLine)
	if !strings.Contains(ready, fmt.Sprint(rcon)) {
		t.Fatalf("RCON startup omitted bound fixed port: %s", ready)
	}
	fact.SetFactorioPipe(server.stdin, 0)
	var exchange string
	hello := func(wantPreview bool) {
		t.Helper()
		id := fact.WriteSoftModCommand("hello", nil)
		previews := 0
		server.until(t, func(line string) bool {
			e, ok := nativeEnvelope(line)
			if !ok {
				return false
			}
			if e.Event == "map-preview" {
				previews++
				var data struct {
					Exchange string `json:"exchange_string"`
				}
				if err := json.Unmarshal(e.Data, &data); err != nil {
					t.Fatal(err)
				}
				exchange = data.Exchange
			}
			if e.ID != id {
				return false
			}
			if !e.OK || e.Command != "hello" || !strings.Contains(string(e.Data), "softmod_version") {
				t.Fatalf("invalid hello: %+v", e)
			}
			return true
		})
		if (previews == 1) != wantPreview || previews > 1 {
			t.Fatalf("hello emitted %d previews, want preview=%v", previews, wantPreview)
		}
	}
	hello(true)
	hello(false)
	wantURL, err := fact.MapPreviewURL(exchange)
	if err != nil {
		t.Fatal(err)
	}
	url, err := fact.ExecuteRCON("map-preview")
	if err != nil || !strings.Contains(url, wantURL) {
		t.Fatalf("RCON map preview: %q %v", url, err)
	}
	hello(false) // manual preview must not rearm automatic announcement
	if _, err := fact.ExecuteRCON("native-save-marker"); err != nil {
		t.Fatal(err)
	}
	quit := func() string {
		t.Helper()
		fact.WriteFact("/quit") // same graceful save-and-quit command as the lifecycle controller
		select {
		case <-server.done:
		case <-time.After(40 * time.Second):
			t.Fatalf("graceful quit timed out:\n%s", server.output())
		}
		if server.err != nil {
			t.Fatalf("graceful quit: %v\n%s", server.err, server.output())
		}
		out := server.output()
		if !strings.Contains(out, "Saving map as") || !strings.Contains(out, "Goodbye") {
			t.Fatalf("missing quit-save completion:\n%s", out)
		}
		return out
	}
	out := quit()
	saves, err := filepath.Glob(filepath.Join(n.root, "saves", "*.zip"))
	if err != nil || len(saves) != 1 {
		t.Fatalf("expected one quit-save, got %v (%v)\n%s", saves, err, out)
	}
	n.run(t, saves[0], map[string]any{"mode": "reload"})
	// The ordinary --start-server path must also handshake after loading the save.
	server = n.start(t, append([]string{"--start-server", saves[0]}, serverArgs...)...)
	server.until(t, isFactorioReadyLine)
	fact.SetFactorioPipe(server.stdin, 0)
	hello(false)
	quit()
	if glob.SoftModVersion != constants.Unknown {
		t.Fatal("test unexpectedly changed production SoftMod state")
	}
}

func TestNativeFactorioEssentials(t *testing.T) {
	n := nativeFixture(t)
	n.run(t, "", map[string]any{"mode": "essentials"})
}
