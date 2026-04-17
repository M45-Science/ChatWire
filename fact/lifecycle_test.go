package fact

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/glob"
)

func newTestLifecycleManager(hooks LifecycleHooks) *lifecycleManager {
	return &lifecycleManager{
		hooks:      hooks,
		phase:      LifecycleStopped,
		phaseSince: time.Now(),
		signalCh:   make(chan struct{}, 8),
		readyCh:    make(chan uint64, 8),
		goodbyeCh:  make(chan uint64, 8),
		exitCh:     make(chan processExitEvent, 8),
		started:    true,
	}
}

func resetLifecycleTestState(t *testing.T) {
	t.Helper()
	lifecycleMu.Lock()
	lifecycle = nil
	lifecycleMu.Unlock()
	glob.FactorioCmd = nil
	glob.FactorioCancel = nil
	glob.FactorioContext = nil
	FactIsRunning = false
	FactorioBooted = false
	FactorioBootedAt = time.Time{}
	FactAutoStart = false
	DoUpdateFactorio = false
	NumPlayers = 0
	QueueReboot = false
	QueueFactReboot = false
	glob.DoRebootCW = false
	glob.ServerRunning = true

	lifecycleStopGraceTimeout = time.Duration(constants.MaxFactorioCloseWait) * 100 * time.Millisecond
	lifecycleStopInterruptTimeout = constants.FactorioStopInterruptTimeout
	lifecycleStopKillTimeout = constants.FactorioStopKillTimeout
	lifecycleStopPollInterval = 100 * time.Millisecond
	lifecyclePlayerWarnDelay = 3 * time.Second
	lifecycleSendQuit = func() { WriteFact("/quit") }
	lifecycleInterruptProcess = func() {
		if glob.FactorioCancel != nil {
			glob.FactorioCancel()
		}
		if glob.FactorioCmd != nil && glob.FactorioCmd.Process != nil {
			_ = glob.FactorioCmd.Process.Signal(os.Interrupt)
		}
	}
	lifecycleKillProcess = func() {
		if glob.FactorioCmd != nil && glob.FactorioCmd.Process != nil {
			_ = glob.FactorioCmd.Process.Kill()
		}
	}
	lifecycleProcessAlive = func() bool {
		return isCurrentFactorioProcessAlive()
	}
}

func writeTestSave(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create save: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("test-map/level.dat0")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	data := make([]byte, 4096)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
}

func TestLifecycleRestartFactorioStartsExactlyOnce(t *testing.T) {
	resetLifecycleTestState(t)

	launches := 0
	lm := newTestLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			launches++
			return nil
		},
	})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.startedAt = time.Now()
	lm.currentGeneration = 1
	lm.syncCompatibilityLocked()

	lm.execute(lifecycleRequest{Request: Request{Kind: ActionRestartFactorio, Reason: "test-restart"}})

	if launches != 1 {
		t.Fatalf("expected one launch, got %d", launches)
	}
	if state := lm.GetState(); state.Phase != LifecycleStarting {
		t.Fatalf("expected phase %q, got %q", LifecycleStarting, state.Phase)
	}
}

func TestLifecycleChangeMapCopiesSaveBeforeLaunch(t *testing.T) {
	resetLifecycleTestState(t)

	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = ""
	cfg.Local.Callsign = "srv"
	cfg.Local.Name = "alpha"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Global.Paths.Folders.Saves = "saves"

	savesDir := filepath.Join(root, cfg.Local.Callsign, cfg.Global.Paths.Folders.FactorioDir, cfg.Global.Paths.Folders.Saves)
	if err := os.MkdirAll(savesDir, 0o755); err != nil {
		t.Fatalf("mkdir saves: %v", err)
	}
	savePath := filepath.Join(savesDir, "candidate.zip")
	writeTestSave(t, savePath)

	launches := 0
	lm := newTestLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			launches++
			newSave := filepath.Join(savesDir, cfg.Local.Name+"_new.zip")
			if _, err := os.Stat(newSave); err != nil {
				t.Fatalf("expected replacement save before launch: %v", err)
			}
			return nil
		},
	})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.startedAt = time.Now()
	lm.currentGeneration = 1
	lm.syncCompatibilityLocked()

	lm.execute(lifecycleRequest{Request: Request{Kind: ActionChangeMap, Reason: "map vote", SaveName: "candidate"}})

	if launches != 1 {
		t.Fatalf("expected one launch, got %d", launches)
	}
	if _, err := os.Stat(filepath.Join(savesDir, cfg.Local.Name+"_new.zip")); err != nil {
		t.Fatalf("expected replacement save: %v", err)
	}
}

func TestLifecyclePriorityPrefersChatWireReboot(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.queue = []lifecycleRequest{
		{Request: Request{Kind: ActionChangeMap, SaveName: "candidate"}},
		{Request: Request{Kind: ActionRestartChatWire, Reason: "chatwire"}},
	}

	req, ok := lm.nextRequest()
	if !ok {
		t.Fatal("expected queued request")
	}
	if req.Kind != ActionRestartChatWire {
		t.Fatalf("expected %q, got %q", ActionRestartChatWire, req.Kind)
	}
}

func TestLifecycleStartupTimeoutQueuesRestart(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStarting
	lm.startedAt = time.Now().Add(-constants.FactorioStartupTimeout - time.Second)
	lm.phaseSince = lm.startedAt
	lm.syncCompatibilityLocked()

	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()

	lm.checkStartupTimeout()

	lm.mu.Lock()
	defer lm.mu.Unlock()
	found := false
	for _, req := range lm.queue {
		if req.Kind == ActionRestartFactorio {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected startup-timeout restart request")
	}
}

func TestLifecycleStopEscalatesToInterrupt(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleStopGraceTimeout = 5 * time.Millisecond
	lifecycleStopInterruptTimeout = 20 * time.Millisecond
	lifecycleStopKillTimeout = 20 * time.Millisecond
	lifecycleStopPollInterval = time.Millisecond

	interrupts := 0
	kills := 0
	alive := true
	lifecycleProcessAlive = func() bool { return alive }
	lifecycleInterruptProcess = func() {
		interrupts++
		alive = false
	}
	lifecycleKillProcess = func() { kills++ }

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = false
	lm.startedAt = time.Now()
	lm.currentGeneration = 1
	lm.syncCompatibilityLocked()

	if err := lm.executeStop("interrupt test"); err != nil {
		t.Fatalf("executeStop returned error: %v", err)
	}
	if interrupts != 1 {
		t.Fatalf("expected 1 interrupt, got %d", interrupts)
	}
	if kills != 0 {
		t.Fatalf("expected 0 kills, got %d", kills)
	}
	if state := lm.GetState(); state.Phase != LifecycleStopped {
		t.Fatalf("expected stopped phase, got %q", state.Phase)
	}
}

func TestLifecycleStopEscalatesToKillOnTimeout(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleStopGraceTimeout = 5 * time.Millisecond
	lifecycleStopInterruptTimeout = 5 * time.Millisecond
	lifecycleStopKillTimeout = 20 * time.Millisecond
	lifecycleStopPollInterval = time.Millisecond

	interrupts := 0
	kills := 0
	alive := true
	lifecycleProcessAlive = func() bool { return alive }
	lifecycleInterruptProcess = func() { interrupts++ }
	lifecycleKillProcess = func() {
		kills++
		alive = false
	}

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = false
	lm.startedAt = time.Now()
	lm.currentGeneration = 1
	lm.syncCompatibilityLocked()

	if err := lm.executeStop("kill test"); err != nil {
		t.Fatalf("executeStop returned error: %v", err)
	}
	if interrupts != 1 {
		t.Fatalf("expected 1 interrupt, got %d", interrupts)
	}
	if kills != 1 {
		t.Fatalf("expected 1 kill, got %d", kills)
	}
	if state := lm.GetState(); state.Phase != LifecycleStopped {
		t.Fatalf("expected stopped phase, got %q", state.Phase)
	}
}

func TestLifecycleWhenEmptyRequestWaitsForNoPlayers(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.currentGeneration = 1
	NumPlayers = 2
	lm.queue = []lifecycleRequest{
		{Request: Request{Kind: ActionRestartFactorio, Reason: "queued", WhenEmpty: true}},
	}
	lm.syncCompatibilityLocked()

	if _, ok := lm.nextRequest(); ok {
		t.Fatal("expected request to remain queued while players are online")
	}
	if !QueueFactReboot {
		t.Fatal("expected compatibility queue flag to stay set")
	}

	NumPlayers = 0
	req, ok := lm.nextRequest()
	if !ok {
		t.Fatal("expected queued request after players left")
	}
	if req.Kind != ActionRestartFactorio {
		t.Fatalf("expected restart-factorio request, got %q", req.Kind)
	}
}

func TestLifecycleHigherPriorityRequestWinsOverQueuedLowerPriority(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error { return nil },
		ExitChatWire:   func(delay bool) {},
	})
	lm.phase = LifecycleRunning
	lm.booted = false
	lm.currentGeneration = 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lifecycleStopGraceTimeout = time.Millisecond
	lifecycleStopInterruptTimeout = time.Millisecond
	lifecycleStopKillTimeout = time.Millisecond
	lifecycleStopPollInterval = time.Millisecond
	lifecycleProcessAlive = func() bool { return false }

	changeDone := make(chan error, 1)
	rebootDone := make(chan error, 1)
	lm.queue = []lifecycleRequest{
		{Request: Request{Kind: ActionChangeMap, SaveName: "candidate", Reason: "change"}, done: changeDone},
		{Request: Request{Kind: ActionRestartChatWire, Reason: "chatwire"}, done: rebootDone},
	}
	if req, ok := lm.nextRequest(); !ok || req.Kind != ActionRestartChatWire {
		t.Fatalf("expected chatwire reboot to run first, got %#v ok=%t", req.Kind, ok)
	}
}

func TestLifecycleStopReturnsErrorIfProcessNeverExits(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleStopGraceTimeout = 5 * time.Millisecond
	lifecycleStopInterruptTimeout = 5 * time.Millisecond
	lifecycleStopKillTimeout = 5 * time.Millisecond
	lifecycleStopPollInterval = time.Millisecond

	interrupts := 0
	kills := 0
	lifecycleProcessAlive = func() bool { return true }
	lifecycleInterruptProcess = func() { interrupts++ }
	lifecycleKillProcess = func() { kills++ }

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.currentGeneration = 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	err := lm.executeStop("timeout test")
	if !errors.Is(err, errors.New("Factorio stop timed out")) && (err == nil || err.Error() != "Factorio stop timed out") {
		t.Fatalf("expected stop timeout error, got %v", err)
	}
	if interrupts != 1 || kills != 1 {
		t.Fatalf("expected interrupt+kill once, got interrupts=%d kills=%d", interrupts, kills)
	}
}
