package fact

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/glob"
)

func newTestLifecycleManager(hooks LifecycleHooks) *lifecycleManager {
	doneCh := make(chan struct{})
	close(doneCh)
	return &lifecycleManager{
		hooks:      hooks,
		phase:      LifecycleStopped,
		phaseSince: time.Now(),
		signalCh:   make(chan struct{}, 8),
		readyCh:    make(chan uint64, 8),
		goodbyeCh:  make(chan uint64, 8),
		exitCh:     make(chan processExitEvent, 8),
		progressCh: make(chan lifecycleProgressEvent, 8),
		healthCh:   make(chan lifecycleHealthEvent, 8),
		started:    true,
		stopCh:     make(chan struct{}),
		doneCh:     doneCh,
	}
}

func resetLifecycleTestState(t *testing.T) {
	t.Helper()
	StopLifecycleManager()
	glob.SetServerRunning(false)
	glob.FactorioCmd = nil
	glob.FactorioCancel = nil
	glob.FactorioContext = nil
	FactIsRunning = false
	FactorioBooted = false
	FactorioBootedAt = time.Time{}
	setAutostartEnabled(false)
	SetUpdateInProgress(false)
	SetModOperationInProgress(false)
	SetNumPlayers(0)
	QueueReboot = false
	QueueFactReboot = false
	glob.DoRebootCW = false
	glob.SetServerRunning(true)

	lifecycleStopGraceTimeout = time.Duration(constants.MaxFactorioCloseWait) * 100 * time.Millisecond
	lifecycleStopIdleTimeout = constants.FactorioStopIdleTimeout
	lifecycleStopSaveTimeout = constants.FactorioStopSaveTimeout
	lifecycleStopInterruptTimeout = constants.FactorioStopInterruptTimeout
	lifecycleStopKillTimeout = constants.FactorioStopKillTimeout
	lifecycleStopPollInterval = 100 * time.Millisecond
	lifecyclePlayerWarnDelay = 3 * time.Second
	lifecycleStartupIdleTimeout = constants.FactorioStartupIdleTimeout
	lifecycleStartupHardTimeout = constants.FactorioStartupHardTimeout
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

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
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

func TestLifecycleUpdateBlocksNonStopRequests(t *testing.T) {
	resetLifecycleTestState(t)

	SetUpdateInProgress(true)
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.queue = []lifecycleRequest{
		{Request: Request{Kind: ActionChangeMap, SaveName: "candidate"}},
		{Request: Request{Kind: ActionMapReset, Reason: "reset"}},
		{Request: Request{Kind: ActionRestartFactorio, Reason: "restart"}},
		{Request: Request{Kind: ActionStart, Reason: "start"}},
	}
	lm.syncCompatibilityLocked()

	if _, ok := lm.nextRequest(); ok {
		t.Fatal("expected update-in-progress to block non-stop requests")
	}
}

func TestLifecycleModOperationBlocksNonStopRequests(t *testing.T) {
	resetLifecycleTestState(t)

	SetModOperationInProgress(true)
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStopped
	lm.queue = []lifecycleRequest{
		{Request: Request{Kind: ActionChangeMap, SaveName: "candidate"}},
		{Request: Request{Kind: ActionMapReset, Reason: "reset"}},
		{Request: Request{Kind: ActionRestartFactorio, Reason: "restart"}},
		{Request: Request{Kind: ActionStart, Reason: "start"}},
	}
	lm.syncCompatibilityLocked()

	if _, ok := lm.nextRequest(); ok {
		t.Fatal("expected mod operation to block non-stop requests")
	}
}

func TestLifecycleUpdateAllowsStopRequest(t *testing.T) {
	resetLifecycleTestState(t)

	SetUpdateInProgress(true)
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.queue = []lifecycleRequest{
		{Request: Request{Kind: ActionChangeMap, SaveName: "candidate"}},
		{Request: Request{Kind: ActionStop, Reason: "update stop"}},
	}
	lm.syncCompatibilityLocked()

	req, ok := lm.nextRequest()
	if !ok {
		t.Fatal("expected stop request to remain runnable during update")
	}
	if req.Kind != ActionStop {
		t.Fatalf("expected stop request during update, got %q", req.Kind)
	}
}

func TestLifecycleQueuedRequestsResumeAfterUpdateFinishes(t *testing.T) {
	resetLifecycleTestState(t)

	SetUpdateInProgress(true)
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStopped
	lm.queue = []lifecycleRequest{
		{Request: Request{Kind: ActionChangeMap, SaveName: "candidate"}},
		{Request: Request{Kind: ActionRestartFactorio, Reason: "restart"}},
	}
	lm.syncCompatibilityLocked()

	if _, ok := lm.nextRequest(); ok {
		t.Fatal("expected no runnable requests while update is active")
	}

	SetUpdateInProgress(false)
	req, ok := lm.nextRequest()
	if !ok {
		t.Fatal("expected queued requests to resume after update completes")
	}
	if req.Kind != ActionChangeMap {
		t.Fatalf("expected change-map to resume first after update, got %q", req.Kind)
	}
}

func TestLifecycleStartupTimeoutQueuesRestart(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStarting
	lifecycleStartupIdleTimeout = 10 * time.Millisecond
	lifecycleStartupHardTimeout = time.Minute
	lm.startedAt = time.Now().Add(-lifecycleStartupIdleTimeout - time.Second)
	lm.phaseSince = lm.startedAt
	lm.lastProgressAt = lm.startedAt
	lm.lastProgressKind = "spawn"
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

func TestLifecycleStartupProgressDefersTimeout(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStarting
	lifecycleStartupIdleTimeout = 15 * time.Millisecond
	lifecycleStartupHardTimeout = time.Minute
	lm.startedAt = time.Now().Add(-time.Second)
	lm.phaseSince = lm.startedAt
	lm.lastProgressAt = time.Now()
	lm.lastProgressKind = "mod-load"
	lm.syncCompatibilityLocked()

	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()

	lm.checkStartupTimeout()

	lm.mu.Lock()
	defer lm.mu.Unlock()
	for _, req := range lm.queue {
		if req.Kind == ActionRestartFactorio {
			t.Fatal("did not expect startup restart while progress is still recent")
		}
	}
}

func TestLifecycleStartupHardTimeoutOverridesProgress(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStarting
	lifecycleStartupIdleTimeout = time.Hour
	lifecycleStartupHardTimeout = 20 * time.Millisecond
	lm.startedAt = time.Now().Add(-time.Second)
	lm.phaseSince = lm.startedAt
	lm.lastProgressAt = time.Now()
	lm.lastProgressKind = "mod-load"
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
		t.Fatal("expected hard-timeout restart request despite recent progress")
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
	SetNumPlayers(2)
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

	SetNumPlayers(0)
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

func TestLifecycleStopSaveProgressExtendsDeadline(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleStopGraceTimeout = 5 * time.Millisecond
	lifecycleStopIdleTimeout = 5 * time.Millisecond
	lifecycleStopSaveTimeout = 40 * time.Millisecond
	lifecycleStopPollInterval = time.Millisecond

	var alive atomic.Bool
	alive.Store(true)
	interrupts := 0
	lifecycleProcessAlive = func() bool { return alive.Load() }
	lifecycleInterruptProcess = func() { interrupts++ }

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = false
	lm.currentGeneration = 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	go func() {
		time.Sleep(3 * time.Millisecond)
		lm.progressCh <- lifecycleProgressEvent{generation: 1, kind: "save", at: time.Now()}
		time.Sleep(10 * time.Millisecond)
		alive.Store(false)
	}()

	if err := lm.executeStop("save progress"); err != nil {
		t.Fatalf("expected stop to succeed with save progress, got %v", err)
	}
	if interrupts != 0 {
		t.Fatalf("expected no interrupt while save progress extends deadline, got %d", interrupts)
	}
}

func TestLifecycleRestartChatWireExitsOnlyAfterStop(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleStopGraceTimeout = time.Millisecond
	lifecycleStopInterruptTimeout = time.Millisecond
	lifecycleStopKillTimeout = time.Millisecond
	lifecycleStopPollInterval = time.Millisecond
	lifecycleProcessAlive = func() bool { return false }

	exitCalls := 0
	var lm *lifecycleManager
	lm = newTestLifecycleManager(LifecycleHooks{
		ExitChatWire: func(delay bool) {
			exitCalls++
			state := lm.GetState()
			if state.Phase != LifecycleStopped {
				t.Fatalf("expected stopped before chatwire exit, got %q", state.Phase)
			}
		},
	})
	lm.phase = LifecycleRunning
	lm.currentGeneration = 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lm.execute(lifecycleRequest{Request: Request{Kind: ActionRestartChatWire, Reason: "chatwire"}})

	if exitCalls != 1 {
		t.Fatalf("expected one chatwire exit call, got %d", exitCalls)
	}
}

func TestLifecycleRunAutoStartsWithinHours(t *testing.T) {
	resetLifecycleTestState(t)

	launchCh := make(chan struct{}, 1)
	StartLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			select {
			case launchCh <- struct{}{}:
			default:
			}
			glob.SetServerRunning(false)
			return nil
		},
		WithinHours: func() bool { return true },
	})
	defer func() {
		glob.SetServerRunning(false)
		StopLifecycleManager()
	}()

	setAutostartEnabled(true)
	waitForCondition(t, 200*time.Millisecond, func() bool {
		select {
		case <-launchCh:
			return true
		default:
			return false
		}
	})
}

func TestLifecycleRunDoesNotAutoStartOutsideHours(t *testing.T) {
	resetLifecycleTestState(t)

	launches := 0
	StartLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			launches++
			return nil
		},
		WithinHours: func() bool { return false },
	})
	defer func() {
		glob.SetServerRunning(false)
		StopLifecycleManager()
	}()

	setAutostartEnabled(true)
	time.Sleep(50 * time.Millisecond)
	if launches != 0 {
		t.Fatalf("expected no autostart launches outside hours, got %d", launches)
	}
}

func TestLifecycleRunDoesNotAutoStartDuringUpdate(t *testing.T) {
	resetLifecycleTestState(t)

	launches := 0
	SetUpdateInProgress(true)
	StartLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			launches++
			return nil
		},
		WithinHours: func() bool { return true },
	})
	defer func() {
		glob.SetServerRunning(false)
		StopLifecycleManager()
	}()

	setAutostartEnabled(true)
	time.Sleep(50 * time.Millisecond)
	if launches != 0 {
		t.Fatalf("expected no autostart launches during update, got %d", launches)
	}
}

func TestLifecycleRunDoesNotAutoStartDuringModOperation(t *testing.T) {
	resetLifecycleTestState(t)

	launches := 0
	SetModOperationInProgress(true)
	StartLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			launches++
			return nil
		},
		WithinHours: func() bool { return true },
	})
	defer func() {
		glob.SetServerRunning(false)
		StopLifecycleManager()
	}()

	setAutostartEnabled(true)
	time.Sleep(50 * time.Millisecond)
	if launches != 0 {
		t.Fatalf("expected no autostart launches during mod operation, got %d", launches)
	}
}

func TestLifecycleExitBeforeReadyLeavesStopped(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStarting
	lm.currentGeneration = 7
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lm.handleExitEvent(processExitEvent{generation: 7, err: nil})
	lm.handleReadyEvent(7)

	state := lm.GetState()
	if state.Phase != LifecycleStopped {
		t.Fatalf("expected stopped after exit-before-ready, got %q", state.Phase)
	}
	if state.Booted {
		t.Fatal("expected booted=false after exit-before-ready")
	}
}

func TestLifecycleReadyThenExitEndsStopped(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleStarting
	lm.currentGeneration = 8
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lm.handleReadyEvent(8)
	if state := lm.GetState(); state.Phase != LifecycleRunning || !state.Booted {
		t.Fatalf("expected running booted state after ready, got phase=%q booted=%t", state.Phase, state.Booted)
	}

	lm.handleExitEvent(processExitEvent{generation: 8, err: nil})
	state := lm.GetState()
	if state.Phase != LifecycleStopped || state.Booted {
		t.Fatalf("expected stopped after ready-then-exit, got phase=%q booted=%t", state.Phase, state.Booted)
	}
}

func TestLifecycleDuplicateGoodbyeDoesNotChangeRunningState(t *testing.T) {
	resetLifecycleTestState(t)

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.currentGeneration = 9
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lm.handleGoodbyeEvent(9)
	lm.handleGoodbyeEvent(9)

	state := lm.GetState()
	if state.Phase != LifecycleRunning || !state.Booted {
		t.Fatalf("expected running state after duplicate goodbye without exit, got phase=%q booted=%t", state.Phase, state.Booted)
	}
}

func TestDoChangeMapAfterStopMissingSourceReturnsError(t *testing.T) {
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

	err := doChangeMapAfterStop("missing")
	if err == nil || err.Error() != "an error occurred when attempting to open the selected save" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoChangeMapAfterStopReplacementRemoveFailureReturnsError(t *testing.T) {
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
	writeTestSave(t, filepath.Join(savesDir, "candidate.zip"))

	targetDir := filepath.Join(savesDir, cfg.Local.Name+"_new.zip")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}

	err := doChangeMapAfterStop("candidate")
	if err == nil || err.Error() != "an error occurred when attempting to remove the existing replacement save" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLifecycleChangeMapLaunchFailureLeavesStopped(t *testing.T) {
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
	writeTestSave(t, filepath.Join(savesDir, "candidate.zip"))

	lm := newTestLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			return errors.New("launch failed")
		},
	})
	lm.phase = LifecycleRunning
	lm.booted = false
	lm.currentGeneration = 1
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()
	lifecycleProcessAlive = func() bool { return false }
	lifecycleStopGraceTimeout = time.Millisecond
	lifecycleStopInterruptTimeout = time.Millisecond
	lifecycleStopKillTimeout = time.Millisecond
	lifecycleStopPollInterval = time.Millisecond

	lm.execute(lifecycleRequest{Request: Request{Kind: ActionChangeMap, Reason: "map", SaveName: "candidate"}})

	if _, err := os.Stat(filepath.Join(savesDir, cfg.Local.Name+"_new.zip")); err != nil {
		t.Fatalf("expected prepared replacement save even when launch fails: %v", err)
	}
	state := lm.GetState()
	if state.Phase != LifecycleStopped {
		t.Fatalf("expected stopped phase after launch failure, got %q", state.Phase)
	}
	if state.LastError != "launch failed" {
		t.Fatalf("expected last error to capture launch failure, got %q", state.LastError)
	}
}

func TestLifecycleExecuteStartFailsDuringUpdate(t *testing.T) {
	resetLifecycleTestState(t)

	SetUpdateInProgress(true)
	lm := newTestLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			t.Fatal("launch should not be called while update is in progress")
			return nil
		},
	})

	err := lm.executeStart("update guarded start")
	if err == nil || err.Error() != "factorio update or mod operation is in progress" {
		t.Fatalf("expected update guard error, got %v", err)
	}
}

func TestLifecycleExecuteStartFailsDuringModOperation(t *testing.T) {
	resetLifecycleTestState(t)

	SetModOperationInProgress(true)
	lm := newTestLifecycleManager(LifecycleHooks{
		LaunchFactorio: func(generation uint64) error {
			t.Fatal("launch should not be called while mod operation is in progress")
			return nil
		},
	})

	err := lm.executeStart("mod guarded start")
	if err == nil || err.Error() != "factorio update or mod operation is in progress" {
		t.Fatalf("expected mod-operation guard error, got %v", err)
	}
}

func TestLifecycleHealthEventQueuesRestartWhenProcessAlive(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleProcessAlive = func() bool { return true }
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.currentGeneration = 12
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lm.handleHealthEvent(lifecycleHealthEvent{
		generation: 12,
		kind:       "stdin-broken-pipe",
		err:        "broken pipe",
		at:         time.Now(),
	})

	lm.mu.Lock()
	defer lm.mu.Unlock()
	if !lm.healthRestartQueued {
		t.Fatal("expected health restart to be queued")
	}
	found := false
	for _, req := range lm.queue {
		if req.Kind == ActionRestartFactorio {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected restart-factorio request after health issue")
	}
}

func TestLifecycleHealthEventFinalizesStoppedWhenProcessDead(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleProcessAlive = func() bool { return false }
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.currentGeneration = 13
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lm.handleHealthEvent(lifecycleHealthEvent{
		generation: 13,
		kind:       "stdout-closed",
		err:        "closed",
		at:         time.Now(),
	})

	state := lm.GetState()
	if state.Phase != LifecycleStopped {
		t.Fatalf("expected stopped after dead-process health event, got %q", state.Phase)
	}
}

func TestLifecycleReconcileProcessHealthStopsMissingProcess(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleProcessAlive = func() bool { return false }
	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.currentGeneration = 14
	lm.currentPID = 1234
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lm.reconcileProcessHealth()

	state := lm.GetState()
	if state.Phase != LifecycleStopped {
		t.Fatalf("expected stopped after health reconciliation, got %q", state.Phase)
	}
}

func TestLifecycleReconcileProcessHealthQueuesRestartOnMissingPipe(t *testing.T) {
	resetLifecycleTestState(t)

	lifecycleProcessAlive = func() bool { return true }
	PipeLock.Lock()
	Pipe = nil
	PipeLock.Unlock()

	lm := newTestLifecycleManager(LifecycleHooks{})
	lm.phase = LifecycleRunning
	lm.booted = true
	lm.currentGeneration = 15
	lm.currentPID = 4321
	lm.startedAt = time.Now()
	lm.syncCompatibilityLocked()

	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()

	lm.reconcileProcessHealth()
	lm.drainAsyncEvents()

	lm.mu.Lock()
	defer lm.mu.Unlock()
	if !lm.healthRestartQueued {
		t.Fatal("expected health restart queued when stdin pipe is missing")
	}
}
