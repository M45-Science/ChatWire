package fact

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

type LifecyclePhase string

const (
	LifecycleStopped  LifecyclePhase = "stopped"
	LifecycleStarting LifecyclePhase = "starting"
	LifecycleRunning  LifecyclePhase = "running"
	LifecycleStopping LifecyclePhase = "stopping"
)

type ActionKind string

const (
	ActionStart           ActionKind = "start"
	ActionStop            ActionKind = "stop"
	ActionRestartFactorio ActionKind = "restart-factorio"
	ActionRestartChatWire ActionKind = "restart-chatwire"
	ActionChangeMap       ActionKind = "change-map"
	ActionMapReset        ActionKind = "map-reset"
)

type Request struct {
	Kind              ActionKind
	Reason            string
	SaveName          string
	ForceChatWireExit bool
	RequestID         string
	WhenEmpty         bool
}

type State struct {
	Phase         LifecyclePhase
	PID           int
	Booted        bool
	PendingAction string
	Since         time.Time
	LastError     string
}

type LifecycleHooks struct {
	LaunchFactorio func(generation uint64) error
	WithinHours    func() bool
	ExitChatWire   func(delay bool)
}

type lifecycleRequest struct {
	Request
	done       chan error
	acceptedAt time.Time
}

type processExitEvent struct {
	generation uint64
	err        error
}

type lifecycleProgressEvent struct {
	generation uint64
	kind       string
	detail     string
	at         time.Time
}

const lifecycleOptionalProgressDelay = 5 * time.Second

type lifecycleHealthEvent struct {
	generation uint64
	kind       string
	err        string
	at         time.Time
}

type lifecycleManager struct {
	mu                  sync.Mutex
	hooks               LifecycleHooks
	phase               LifecyclePhase
	phaseSince          time.Time
	booted              bool
	currentPID          int
	currentGeneration   uint64
	currentAction       string
	lastError           string
	startedAt           time.Time
	readyAt             time.Time
	queue               []lifecycleRequest
	signalCh            chan struct{}
	readyCh             chan uint64
	goodbyeCh           chan uint64
	exitCh              chan processExitEvent
	progressCh          chan lifecycleProgressEvent
	healthCh            chan lifecycleHealthEvent
	started             bool
	shutdownRequested   bool
	healthRestartQueued bool
	operationToken      string
	operationKind       ActionKind
	operationSaveName   string
	lastProgressAt      time.Time
	lastProgressKind    string
	stopCh              chan struct{}
	doneCh              chan struct{}
}

var (
	lifecycleMu sync.Mutex
	lifecycle   *lifecycleManager
	reqCounter  atomic.Uint64

	lifecycleStopGraceTimeout     = time.Duration(constants.MaxFactorioCloseWait) * 100 * time.Millisecond
	lifecycleStopIdleTimeout      = constants.FactorioStopIdleTimeout
	lifecycleStopSaveTimeout      = constants.FactorioStopSaveTimeout
	lifecycleStopInterruptTimeout = constants.FactorioStopInterruptTimeout
	lifecycleStopKillTimeout      = constants.FactorioStopKillTimeout
	lifecycleStopPollInterval     = 100 * time.Millisecond
	lifecyclePlayerWarnDelay      = 3 * time.Second
	lifecycleStartupIdleTimeout   = constants.FactorioStartupIdleTimeout
	lifecycleStartupHardTimeout   = constants.FactorioStartupHardTimeout
	lifecycleSendQuit             = func() { WriteFact("/quit") }
	lifecycleInterruptProcess     = func() {
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
)

func StartLifecycleManager(hooks LifecycleHooks) {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()

	if lifecycle != nil {
		return
	}

	lifecycle = &lifecycleManager{
		hooks:      hooks,
		phase:      LifecycleStopped,
		phaseSince: time.Now(),
		signalCh:   make(chan struct{}, 1),
		readyCh:    make(chan uint64, 8),
		goodbyeCh:  make(chan uint64, 8),
		exitCh:     make(chan processExitEvent, 8),
		progressCh: make(chan lifecycleProgressEvent, 32),
		healthCh:   make(chan lifecycleHealthEvent, 16),
		started:    true,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
	lifecycle.syncCompatibilityLocked()

	go lifecycle.run()
}

func StopLifecycleManager() {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()
	if lm == nil {
		return
	}

	lm.mu.Lock()
	if !lm.shutdownRequested {
		lm.shutdownRequested = true
		close(lm.stopCh)
	}
	doneCh := lm.doneCh
	lm.mu.Unlock()

	<-doneCh

	lifecycleMu.Lock()
	if lifecycle == lm {
		lifecycle = nil
	}
	lifecycleMu.Unlock()
}

func SubmitLifecycleRequest(req Request) error {
	_, err := submitLifecycleRequest(req, false)
	return err
}

func submitLifecycleRequestAndWait(req Request) error {
	_, err := submitLifecycleRequest(req, true)
	return err
}

func submitLifecycleRequest(req Request, wait bool) (State, error) {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()

	if lm == nil {
		return State{}, errors.New("lifecycle manager is not running")
	}

	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("req-%d", reqCounter.Add(1))
	}

	lr := lifecycleRequest{
		Request:    req,
		acceptedAt: time.Now(),
	}
	if wait {
		lr.done = make(chan error, 1)
	}

	lm.mu.Lock()
	lm.queue = append(lm.queue, lr)
	lm.syncCompatibilityLocked()
	state := lm.currentStateLocked()
	lm.mu.Unlock()

	cwlog.DoLogCW("lifecycle: request accepted kind=%s id=%s when_empty=%t reason=%q", req.Kind, req.RequestID, req.WhenEmpty, req.Reason)
	lm.signal()

	if lr.done != nil {
		err := <-lr.done
		return lm.GetState(), err
	}
	return state, nil
}

func GetLifecycleState() State {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()
	if lm == nil {
		return State{Phase: LifecycleStopped, Since: time.Now()}
	}
	return lm.GetState()
}

func (lm *lifecycleManager) GetState() State {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.currentStateLocked()
}

func NotifyFactorioReady() {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()
	if lm == nil {
		return
	}
	gen := lm.getCurrentGeneration()
	select {
	case lm.readyCh <- gen:
	default:
	}
	lm.signal()
}

func NotifyFactorioGoodbye() {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()
	if lm == nil {
		return
	}
	gen := lm.getCurrentGeneration()
	select {
	case lm.goodbyeCh <- gen:
	default:
	}
	lm.signal()
}

func NotifyFactorioProcessExit(generation uint64, err error) {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()
	if lm == nil {
		return
	}
	select {
	case lm.exitCh <- processExitEvent{generation: generation, err: err}:
	default:
	}
	lm.signal()
}

func NotifyFactorioProgress(kind, detail string) {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()
	if lm == nil || kind == "" {
		return
	}
	evt := lifecycleProgressEvent{
		generation: lm.getCurrentGeneration(),
		kind:       kind,
		detail:     detail,
		at:         time.Now(),
	}
	select {
	case lm.progressCh <- evt:
	default:
	}
	lm.signal()
}

func NotifyFactorioHealth(kind string, err error) {
	lifecycleMu.Lock()
	lm := lifecycle
	lifecycleMu.Unlock()
	if lm == nil || kind == "" {
		return
	}
	evt := lifecycleHealthEvent{
		generation: lm.getCurrentGeneration(),
		kind:       kind,
		at:         time.Now(),
	}
	if err != nil {
		evt.err = err.Error()
	}
	select {
	case lm.healthCh <- evt:
	default:
	}
	lm.signal()
}

func WaitForLifecycleStop(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if GetLifecycleState().Phase == LifecycleStopped {
			return true
		}
		if timeout > 0 && time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (lm *lifecycleManager) signal() {
	select {
	case lm.signalCh <- struct{}{}:
	default:
	}
}

func (lm *lifecycleManager) getCurrentGeneration() uint64 {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.currentGeneration
}

func (lm *lifecycleManager) currentStateLocked() State {
	pending := ""
	if len(lm.queue) > 0 {
		pending = string(lm.queue[0].Kind)
	}
	if lm.currentAction != "" {
		pending = lm.currentAction
	}
	return State{
		Phase:         lm.phase,
		PID:           lm.currentPID,
		Booted:        lm.booted,
		PendingAction: pending,
		Since:         lm.phaseSince,
		LastError:     lm.lastError,
	}
}

func (lm *lifecycleManager) syncCompatibilityLocked() {
	FactIsRunning = lm.phase == LifecycleStarting || lm.phase == LifecycleRunning || lm.phase == LifecycleStopping
	FactorioBooted = lm.booted
	if lm.phase == LifecycleStopped {
		FactorioBootedAt = time.Time{}
		clearOnlinePlayers()
	} else if !lm.startedAt.IsZero() {
		FactorioBootedAt = lm.startedAt
	}

	QueueReboot = false
	QueueFactReboot = false
	glob.DoRebootCW = false
	for _, req := range lm.queue {
		switch req.Kind {
		case ActionRestartChatWire:
			if req.WhenEmpty {
				QueueReboot = true
			}
			glob.DoRebootCW = true
		case ActionRestartFactorio:
			if req.WhenEmpty {
				QueueFactReboot = true
			}
		}
	}
}

func (lm *lifecycleManager) run() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	defer close(lm.doneCh)

	for {
		lm.drainAsyncEvents()
		lm.reconcileProcessHealth()
		lm.checkStartupTimeout()

		if req, ok := lm.nextRequest(); ok {
			lm.execute(req)
			continue
		}

		if lm.shouldAutoStart() {
			lm.execute(lifecycleRequest{
				Request: Request{
					Kind:      ActionStart,
					Reason:    "auto-start",
					RequestID: fmt.Sprintf("auto-%d", reqCounter.Add(1)),
				},
				acceptedAt: time.Now(),
			})
			continue
		}

		select {
		case <-lm.stopCh:
			return
		case <-lm.signalCh:
		case <-ticker.C:
		}
	}
}

func (lm *lifecycleManager) shouldAutoStart() bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.phase == LifecycleStopped &&
		AutostartEnabled() &&
		!UpdateInProgress() &&
		!ModOperationInProgress() &&
		lm.hooks.LaunchFactorio != nil &&
		(lm.hooks.WithinHours == nil || lm.hooks.WithinHours())
}

func (lm *lifecycleManager) nextRequest() (lifecycleRequest, bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	bestIdx := -1
	bestScore := -1
	for i, req := range lm.queue {
		if !lm.requestRunnableLocked(req) {
			continue
		}
		score := actionPriority(req.Kind)
		if bestIdx == -1 || score > bestScore {
			bestIdx = i
			bestScore = score
		}
	}
	if bestIdx == -1 {
		return lifecycleRequest{}, false
	}
	req := lm.queue[bestIdx]
	lm.queue = append(lm.queue[:bestIdx], lm.queue[bestIdx+1:]...)
	lm.currentAction = string(req.Kind)
	lm.syncCompatibilityLocked()
	return req, true
}

func (lm *lifecycleManager) requestRunnableLocked(req lifecycleRequest) bool {
	if (UpdateInProgress() || ModOperationInProgress()) &&
		req.Kind != ActionStop &&
		!(req.Kind == ActionRestartChatWire && req.ForceChatWireExit) {
		return false
	}
	if req.WhenEmpty && lm.phase != LifecycleStopped && NumPlayersCurrent() > 0 {
		return false
	}
	return true
}

func actionPriority(kind ActionKind) int {
	switch kind {
	case ActionRestartChatWire:
		return 50
	case ActionChangeMap, ActionMapReset:
		return 40
	case ActionRestartFactorio:
		return 30
	case ActionStop:
		return 20
	case ActionStart:
		return 10
	default:
		return 0
	}
}

func lifecycleOperationTitle(kind ActionKind) string {
	switch kind {
	case ActionStart:
		return "Starting Factorio"
	case ActionStop:
		return "Stopping Factorio"
	case ActionRestartFactorio:
		return "Restarting Factorio"
	case ActionRestartChatWire:
		return "Restarting ChatWire"
	case ActionChangeMap:
		return "Changing Map"
	case ActionMapReset:
		return "Resetting Map"
	default:
		return "Operation"
	}
}

func lifecycleOperationDescriptionForKind(kind ActionKind, saveName string) string {
	switch kind {
	case ActionStart:
		return StatusStartingFactorio()
	case ActionStop:
		return StatusStoppingFactorio()
	case ActionRestartFactorio:
		return StatusRestartingFactorio()
	case ActionRestartChatWire:
		return StatusRestartingChatWire()
	case ActionChangeMap:
		return StatusChangingMap(saveName)
	case ActionMapReset:
		return StatusResettingMap()
	default:
		return "Operation in progress."
	}
}

func shouldAnnounceOperationStart(kind ActionKind) bool {
	return true
}

func (lm *lifecycleManager) beginOperation(req lifecycleRequest) {
	title := lifecycleOperationTitle(req.Kind)
	description := lifecycleOperationDescriptionForKind(req.Kind, req.SaveName)
	token := BeginOperation(title, description)

	lm.mu.Lock()
	lm.operationToken = token
	lm.operationKind = req.Kind
	lm.operationSaveName = req.SaveName
	lm.mu.Unlock()

	if shouldAnnounceOperationStart(req.Kind) {
		AnnounceOperationNow(token, title, description, glob.COLOR_CYAN)
	}
}

func (lm *lifecycleManager) refreshOperationAnnouncement() {
	lm.mu.Lock()
	token := lm.operationToken
	kind := lm.operationKind
	saveName := lm.operationSaveName
	lm.mu.Unlock()
	if token == "" || kind == "" {
		return
	}
	UpdateOperation(token, lifecycleOperationTitle(kind), lifecycleOperationDescriptionForKind(kind, saveName), glob.COLOR_CYAN)
}

func (lm *lifecycleManager) finishOperationSuccess(description string, color int) {
	lm.mu.Lock()
	token := lm.operationToken
	title := lifecycleOperationTitle(lm.operationKind)
	lm.operationToken = ""
	lm.operationKind = ""
	lm.operationSaveName = ""
	lm.mu.Unlock()
	CompleteOperation(token, title, description, color)
}

func (lm *lifecycleManager) finishOperationError(err error) {
	if err == nil {
		return
	}
	lm.mu.Lock()
	token := lm.operationToken
	title := lifecycleOperationTitle(lm.operationKind)
	lm.operationToken = ""
	lm.operationKind = ""
	lm.operationSaveName = ""
	lm.mu.Unlock()
	FailOperation(token, title, err.Error(), glob.COLOR_RED)
}

func (lm *lifecycleManager) updateOperationProgress(description string) {
	lm.mu.Lock()
	token := lm.operationToken
	title := lifecycleOperationTitle(lm.operationKind)
	lm.mu.Unlock()
	UpdateOperationProgress(token, title, description, glob.COLOR_CYAN)
}

func (lm *lifecycleManager) updateOperationProgressDelayed(description string, delay time.Duration) {
	lm.mu.Lock()
	token := lm.operationToken
	title := lifecycleOperationTitle(lm.operationKind)
	lm.mu.Unlock()
	UpdateOperationProgressDelayed(token, title, description, glob.COLOR_CYAN, delay)
}

func (lm *lifecycleManager) updateOperationProgressDelayedWithReminder(description, reminder string, delay time.Duration) {
	lm.mu.Lock()
	token := lm.operationToken
	title := lifecycleOperationTitle(lm.operationKind)
	lm.mu.Unlock()
	UpdateOperationProgressDelayedWithReminder(token, title, description, reminder, glob.COLOR_CYAN, delay)
}

func (lm *lifecycleManager) execute(req lifecycleRequest) {
	started := time.Now()
	var err error

	lm.beginOperation(req)

	switch req.Kind {
	case ActionStart:
		err = lm.executeStart(req.Reason)
	case ActionStop:
		err = lm.executeStop(req.Reason)
	case ActionRestartFactorio:
		err = lm.executeStop(req.Reason)
		if err == nil {
			err = lm.executeStart(req.Reason)
		}
	case ActionRestartChatWire:
		err = lm.executeStop(req.Reason)
		if err == nil && lm.hooks.ExitChatWire != nil {
			cwlog.DoLogCW("lifecycle: follow-up action executed kind=%s", req.Kind)
			lm.hooks.ExitChatWire(req.ForceChatWireExit)
		}
	case ActionChangeMap:
		err = lm.executeStop(req.Reason)
		if err == nil {
			err = doChangeMapAfterStop(req.SaveName)
		}
		if err == nil {
			cwlog.DoLogCW("lifecycle: change-map prepare completed save=%s elapsed=%v", req.SaveName, time.Since(started).Round(time.Millisecond))
			SetAutolaunch(true, false)
			err = lm.executeStart(req.Reason)
		}
	case ActionMapReset:
		err = lm.executeStop(req.Reason)
		if err == nil {
			err = mapResetAfterStop(false)
		}
		if err == nil {
			cwlog.DoLogCW("lifecycle: follow-up action executed kind=%s elapsed=%v", req.Kind, time.Since(started).Round(time.Millisecond))
			SetAutolaunch(true, false)
			err = lm.executeStart(req.Reason)
		}
	default:
		err = fmt.Errorf("unknown lifecycle action: %s", req.Kind)
	}

	lm.mu.Lock()
	lm.currentAction = ""
	if err != nil {
		lm.lastError = err.Error()
	}
	lm.syncCompatibilityLocked()
	lm.mu.Unlock()

	if err != nil {
		lm.finishOperationError(err)
	} else {
		switch req.Kind {
		case ActionStop:
			lm.finishOperationSuccess(StatusFactorioOffline(), glob.COLOR_RED)
		}
	}

	if req.done != nil {
		req.done <- err
	}
}

func (lm *lifecycleManager) executeStart(reason string) error {
	lm.mu.Lock()
	if lm.phase == LifecycleRunning || lm.phase == LifecycleStarting {
		lm.currentAction = ""
		lm.mu.Unlock()
		return nil
	}
	if UpdateInProgress() || ModOperationInProgress() {
		lm.currentAction = ""
		lm.mu.Unlock()
		return errors.New("factorio update or mod operation is in progress")
	}
	if lm.hooks.WithinHours != nil && !lm.hooks.WithinHours() {
		lm.currentAction = ""
		lm.mu.Unlock()
		return errors.New("outside allowed play hours")
	}
	lm.phase = LifecycleStarting
	lm.phaseSince = time.Now()
	lm.startedAt = lm.phaseSince
	lm.lastProgressAt = lm.phaseSince
	lm.lastProgressKind = "spawn"
	lm.readyAt = time.Time{}
	lm.booted = false
	lm.currentPID = 0
	lm.currentGeneration++
	gen := lm.currentGeneration
	lm.syncCompatibilityLocked()
	lm.mu.Unlock()

	cwlog.DoLogCW("lifecycle: start initiated generation=%d reason=%q", gen, reason)
	started := time.Now()
	if err := lm.hooks.LaunchFactorio(gen); err != nil {
		lm.mu.Lock()
		lm.phase = LifecycleStopped
		lm.phaseSince = time.Now()
		lm.startedAt = time.Time{}
		lm.currentPID = 0
		lm.booted = false
		lm.lastError = err.Error()
		lm.syncCompatibilityLocked()
		lm.mu.Unlock()
		return err
	}

	lm.mu.Lock()
	if glob.FactorioCmd != nil && glob.FactorioCmd.Process != nil {
		lm.currentPID = glob.FactorioCmd.Process.Pid
	}
	lm.syncCompatibilityLocked()
	lm.mu.Unlock()

	cwlog.DoLogCW("lifecycle: process spawned generation=%d pid=%d spawn_elapsed=%v", gen, lm.GetState().PID, time.Since(started).Round(time.Millisecond))
	return nil
}

func (lm *lifecycleManager) executeStop(reason string) error {
	lm.mu.Lock()
	if lm.phase == LifecycleStopped {
		lm.currentAction = ""
		lm.syncCompatibilityLocked()
		lm.mu.Unlock()
		return nil
	}
	lm.phase = LifecycleStopping
	lm.phaseSince = time.Now()
	lm.lastProgressAt = lm.phaseSince
	lm.lastProgressKind = "stop-requested"
	gen := lm.currentGeneration
	booted := lm.booted
	lm.syncCompatibilityLocked()
	lm.mu.Unlock()

	if reason == "" {
		reason = "Server quitting."
	}

	cwlog.DoLogCW("lifecycle: stop initiated generation=%d reason=%q", gen, reason)
	glob.RelaunchThrottle = 0
	glob.ResetNoResponseCount()

	if booted && NumPlayersCurrent() > 0 {
		FactChat("[color=red]" + reason + "[/color]")
		FactChat("[color=green]" + reason + "[/color]")
		FactChat("[color=blue]" + reason + "[/color]")
		FactChat("[color=white]" + reason + "[/color]")
		FactChat("[color=black]" + reason + "[/color]")
		time.Sleep(lifecyclePlayerWarnDelay)
	}

	if booted && FactIsRunning {
		cwlog.DoLogCW("lifecycle: /quit sent generation=%d", gen)
		cwlog.DoLogGame("Quitting Factorio...")
		lifecycleSendQuit()
	}

	if lm.waitForStop(gen, lifecycleStopGraceTimeout, false) {
		return nil
	}

	cwlog.DoLogCW("lifecycle: interrupt sent generation=%d", gen)
	lifecycleInterruptProcess()
	if lm.waitForStop(gen, lifecycleStopInterruptTimeout, true) {
		return nil
	}

	cwlog.DoLogCW("lifecycle: kill sent generation=%d", gen)
	lifecycleKillProcess()
	if lm.waitForStop(gen, lifecycleStopKillTimeout, true) {
		return nil
	}

	lm.finalizeStopped(gen, errors.New("Factorio stop timed out"), true)
	return errors.New("Factorio stop timed out")
}

func (lm *lifecycleManager) waitForStop(generation uint64, timeout time.Duration, allowGoodbye bool) bool {
	deadline := time.Now().Add(timeout)
	goodbyeSeen := false

	for {
		lm.mu.Lock()
		lastProgressAt := lm.lastProgressAt
		lastProgressKind := lm.lastProgressKind
		lm.mu.Unlock()
		if lastProgressKind == "save" && !lastProgressAt.IsZero() {
			if saveDeadline := lastProgressAt.Add(lifecycleStopSaveTimeout); saveDeadline.After(deadline) {
				deadline = saveDeadline
			}
		} else if shouldExtendStopDeadline(lastProgressKind) && !lastProgressAt.IsZero() {
			if progressDeadline := lastProgressAt.Add(lifecycleStopIdleTimeout); progressDeadline.After(deadline) {
				deadline = progressDeadline
			}
		}
		if allowGoodbye && goodbyeSeen && !lifecycleProcessAlive() {
			lm.finalizeStopped(generation, nil, true)
			return true
		}
		if !lifecycleProcessAlive() {
			lm.finalizeStopped(generation, nil, true)
			return true
		}
		if time.Now().After(deadline) {
			return false
		}

		select {
		case evt := <-lm.exitCh:
			if evt.generation != generation {
				lm.handleExitEvent(evt)
				continue
			}
			lm.handleExitEvent(evt)
			return true
		case gen := <-lm.goodbyeCh:
			if gen == generation {
				goodbyeSeen = true
			}
		case gen := <-lm.readyCh:
			if gen == generation {
				lm.handleReadyEvent(gen)
			}
		case evt := <-lm.progressCh:
			lm.handleProgressEvent(evt)
		case <-time.After(lifecycleStopPollInterval):
		}
	}
}

func shouldExtendStopDeadline(kind string) bool {
	switch kind {
	case "", "stop-requested", "spawn":
		return false
	default:
		return true
	}
}

func (lm *lifecycleManager) drainAsyncEvents() {
	for {
		select {
		case gen := <-lm.readyCh:
			lm.handleReadyEvent(gen)
		case gen := <-lm.goodbyeCh:
			lm.handleGoodbyeEvent(gen)
		case evt := <-lm.exitCh:
			lm.handleExitEvent(evt)
		case evt := <-lm.progressCh:
			lm.handleProgressEvent(evt)
		case evt := <-lm.healthCh:
			lm.handleHealthEvent(evt)
		default:
			return
		}
	}
}

func (lm *lifecycleManager) handleProgressEvent(evt lifecycleProgressEvent) {
	lm.mu.Lock()
	if evt.generation != lm.currentGeneration || evt.generation == 0 {
		lm.mu.Unlock()
		return
	}
	lm.lastProgressAt = evt.at
	lm.lastProgressKind = evt.kind
	opKind := lm.operationKind
	saveName := lm.operationSaveName
	lm.mu.Unlock()

	cwlog.DoLogCW("lifecycle: progress observed generation=%d kind=%s", evt.generation, evt.kind)

	switch evt.kind {
	case "mod-load":
		lm.updateOperationProgressDelayedWithReminder(StatusLoadingMods(evt.detail), StatusLoadingModsStill(evt.detail), lifecycleOptionalProgressDelay)
	case "map-load":
		if opKind == ActionChangeMap {
			lm.updateOperationProgressDelayedWithReminder(StatusLoadingMap(saveName), StatusLoadingMapStill(saveName), lifecycleOptionalProgressDelay)
		} else {
			lm.updateOperationProgressDelayedWithReminder(StatusLoadingMap(""), StatusLoadingMapStill(""), lifecycleOptionalProgressDelay)
		}
	case "save":
		lm.updateOperationProgressDelayedWithReminder(StatusSavingMap(), StatusSavingMapStill(), lifecycleOptionalProgressDelay)
	case "rcon-ready":
		lm.updateOperationProgressDelayedWithReminder(StatusBringingServerOnline(), StatusBringingServerOnlineStill(), lifecycleOptionalProgressDelay)
	}
}

func (lm *lifecycleManager) handleHealthEvent(evt lifecycleHealthEvent) {
	if evt.generation == 0 {
		return
	}

	shouldSignal := false

	lm.mu.Lock()
	if evt.generation != lm.currentGeneration {
		lm.mu.Unlock()
		return
	}
	if evt.err != "" {
		lm.lastError = evt.err
	}
	phase := lm.phase
	if phase == LifecycleStopping || phase == LifecycleStopped {
		lm.mu.Unlock()
		cwlog.DoLogCW("lifecycle: health event ignored generation=%d phase=%s kind=%s err=%q", evt.generation, phase, evt.kind, evt.err)
		return
	}
	if !lifecycleProcessAlive() {
		lm.mu.Unlock()
		cwlog.DoLogCW("lifecycle: health event observed generation=%d kind=%s err=%q process_alive=false", evt.generation, evt.kind, evt.err)
		lm.finishOperationError(errors.New(evt.kind))
		lm.finalizeStopped(evt.generation, errors.New(evt.kind), true)
		return
	}
	if !lm.healthRestartQueued {
		lm.queue = append(lm.queue, lifecycleRequest{
			Request: Request{
				Kind:      ActionRestartFactorio,
				Reason:    fmt.Sprintf("Factorio health issue detected (%s): %s", evt.kind, evt.err),
				RequestID: fmt.Sprintf("health-%d-%s", evt.generation, evt.kind),
			},
			acceptedAt: time.Now(),
		})
		lm.healthRestartQueued = true
		lm.syncCompatibilityLocked()
		shouldSignal = true
	}
	lm.mu.Unlock()

	cwlog.DoLogCW("lifecycle: health event observed generation=%d kind=%s err=%q process_alive=true", evt.generation, evt.kind, evt.err)
	lm.updateOperationProgress(fmt.Sprintf("Detected a Factorio health issue (%s). A controlled restart has been queued.", evt.kind))
	if shouldSignal {
		lm.signal()
	}
}

func (lm *lifecycleManager) handleReadyEvent(generation uint64) {
	lm.mu.Lock()
	if generation != lm.currentGeneration || lm.phase != LifecycleStarting {
		lm.mu.Unlock()
		return
	}

	lm.phase = LifecycleRunning
	lm.phaseSince = time.Now()
	lm.booted = true
	lm.readyAt = lm.phaseSince
	lm.syncCompatibilityLocked()

	token := lm.operationToken
	kind := lm.operationKind
	saveName := lm.operationSaveName
	readyElapsed := lm.readyAt.Sub(lm.startedAt).Round(time.Millisecond)
	lm.operationToken = ""
	lm.operationKind = ""
	lm.operationSaveName = ""
	lm.mu.Unlock()

	readyMsg := waitForFactorioReadyStatus(factorioReadyVersionTimeout)
	cwlog.DoLogGame(readyMsg)
	glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Ready", readyMsg, glob.COLOR_GREEN))
	UpdateChannelName()
	DoUpdateChannelNameForce()
	glob.CrashLoopCount = 0
	cwlog.DoLogCW("lifecycle: ready observed generation=%d ready_elapsed=%v", generation, readyElapsed)

	switch kind {
	case ActionStart, ActionRestartFactorio:
		CompleteOperation(token, "", "", glob.COLOR_GREEN)
	case ActionChangeMap:
		desc := "Map change complete. Factorio is online."
		if saveName != "" {
			desc = fmt.Sprintf("Map change to %s complete. Factorio is online.", saveName)
		}
		CompleteOperation(token, lifecycleOperationTitle(kind), desc, glob.COLOR_GREEN)
	case ActionMapReset:
		CompleteOperation(token, lifecycleOperationTitle(kind), "Map reset complete. Factorio is online.", glob.COLOR_GREEN)
	}
}

func (lm *lifecycleManager) handleGoodbyeEvent(generation uint64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if generation != lm.currentGeneration {
		return
	}
	cwlog.DoLogCW("lifecycle: goodbye observed generation=%d", generation)
}

func (lm *lifecycleManager) handleExitEvent(evt processExitEvent) {
	if evt.generation == 0 {
		return
	}
	lm.finalizeStopped(evt.generation, evt.err, true)
}

func shouldReportOfflineStatus(opKind ActionKind, exitErr error) bool {
	if exitErr != nil {
		return true
	}
	switch opKind {
	case ActionRestartFactorio, ActionRestartChatWire, ActionChangeMap, ActionMapReset:
		return false
	default:
		return true
	}
}

func (lm *lifecycleManager) finalizeStopped(generation uint64, exitErr error, report bool) {
	lm.mu.Lock()
	if generation != lm.currentGeneration && generation != 0 {
		lm.mu.Unlock()
		return
	}
	if lm.phase == LifecycleStopped {
		lm.mu.Unlock()
		return
	}

	wasStarting := lm.phase == LifecycleStarting
	startedAt := lm.startedAt
	stoppedAfter := time.Since(lm.phaseSince).Round(time.Millisecond)
	opToken := lm.operationToken
	opKind := lm.operationKind
	opSaveName := lm.operationSaveName

	lm.phase = LifecycleStopped
	lm.phaseSince = time.Now()
	lm.booted = false
	lm.startedAt = time.Time{}
	lm.readyAt = time.Time{}
	lm.lastProgressAt = time.Time{}
	lm.lastProgressKind = ""
	lm.healthRestartQueued = false
	lm.operationToken = ""
	lm.operationKind = ""
	lm.operationSaveName = ""
	lm.currentPID = 0
	lm.currentAction = ""
	if exitErr != nil && !strings.Contains(exitErr.Error(), "signal: killed") {
		lm.lastError = exitErr.Error()
	}
	lm.syncCompatibilityLocked()

	PipeLock.Lock()
	Pipe = nil
	PipeLock.Unlock()
	glob.FactorioCmd = nil
	glob.FactorioCancel = nil
	glob.FactorioContext = nil

	UpdateChannelName()
	DoUpdateChannelNameForce()
	if report && shouldReportOfflineStatus(opKind, exitErr) {
		cwlog.DoLogCW("Factorio has closed.")
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Offline", StatusFactorioOffline(), glob.COLOR_RED))
	}

	if wasStarting {
		if !startedAt.IsZero() && time.Since(startedAt) < 2*time.Minute {
			glob.CrashLoopCount++
		} else {
			glob.CrashLoopCount = 0
		}
		glob.LastCrash = time.Now()
		if glob.CrashLoopCount >= 3 {
			SetAutolaunch(false, true)
			msg := fmt.Sprintf("%s-%s: %s: Factorio crashed repeatedly during startup while loading. Moderator attention required, auto-start option disabled.",
				cfg.Global.GroupName, cfg.Local.Callsign, cfg.Local.Name)
			cfg.Local.Options.AutoStart = false
			cfg.WriteLCfg()
			disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, msg)
			cwlog.DoLogCW(msg)
		}
	}

	cwlog.DoLogCW("lifecycle: process exit observed generation=%d stop_elapsed=%v", generation, stoppedAfter)
	lm.mu.Unlock()

	if opToken != "" {
		CancelOperationDelayedProgress(opToken)
	}

	if opToken != "" && exitErr == nil {
		switch opKind {
		case ActionStop:
			CancelOperation(opToken)
		}
	}
	if opToken != "" && (wasStarting || exitErr != nil) {
		desc := "Factorio stopped unexpectedly."
		if wasStarting {
			desc = "Factorio stopped before startup completed."
		}
		if exitErr != nil && !strings.Contains(exitErr.Error(), "signal: killed") {
			desc = exitErr.Error()
		}
		if opKind == ActionChangeMap && opSaveName != "" && wasStarting {
			desc = fmt.Sprintf("Map change to %s did not complete.", opSaveName)
		}
		FailOperation(opToken, lifecycleOperationTitle(opKind), desc, glob.COLOR_RED)
	}
}

func (lm *lifecycleManager) checkStartupTimeout() {
	lm.mu.Lock()
	if lm.phase != LifecycleStarting || lm.startedAt.IsZero() {
		lm.mu.Unlock()
		return
	}
	startedAt := lm.startedAt
	lastProgressAt := lm.lastProgressAt
	lastProgressKind := lm.lastProgressKind
	lm.mu.Unlock()

	now := time.Now()
	if now.Sub(startedAt) <= lifecycleStartupHardTimeout {
		progressRef := startedAt
		if !lastProgressAt.IsZero() {
			progressRef = lastProgressAt
		}
		if now.Sub(progressRef) <= lifecycleStartupIdleTimeout {
			return
		}
		cwlog.DoLogCW("lifecycle: startup idle timeout hit after %v without progress (last=%s)", lifecycleStartupIdleTimeout, lastProgressKind)
	} else {
		cwlog.DoLogCW("lifecycle: startup hard timeout hit after %v", lifecycleStartupHardTimeout)
	}
	lm.finishOperationError(fmt.Errorf("Factorio startup timed out after %v (last progress: %s)", now.Sub(startedAt).Round(time.Second), lastProgressKind))
	_ = SubmitLifecycleRequest(Request{
		Kind:      ActionRestartFactorio,
		Reason:    fmt.Sprintf("Factorio startup timed out after %v (last progress: %s); forcing restart.", now.Sub(startedAt).Round(time.Second), lastProgressKind),
		RequestID: fmt.Sprintf("startup-timeout-%d", reqCounter.Add(1)),
	})
}

func (lm *lifecycleManager) reconcileProcessHealth() {
	lm.mu.Lock()
	phase := lm.phase
	generation := lm.currentGeneration
	currentPID := lm.currentPID
	lm.mu.Unlock()

	if phase == LifecycleStopped || generation == 0 {
		return
	}

	if !lifecycleProcessAlive() {
		cwlog.DoLogCW("lifecycle: process health reconciliation observed missing process generation=%d pid=%d phase=%s", generation, currentPID, phase)
		lm.finalizeStopped(generation, errors.New("factorio process no longer alive"), true)
		return
	}

	if (phase == LifecycleStarting || phase == LifecycleRunning) && !hasFactorioPipe() {
		NotifyFactorioHealth("stdin-missing", errors.New("factorio stdin pipe is not available"))
	}
}

func hasFactorioPipe() bool {
	PipeLock.Lock()
	defer PipeLock.Unlock()
	return Pipe != nil
}

func isCurrentFactorioProcessAlive() bool {
	cmd := glob.FactorioCmd
	if cmd == nil || cmd.Process == nil {
		return false
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", cmd.Process.Pid)); err == nil {
		if s := string(data); s != "" {
			if idx := strings.LastIndex(s, ") "); idx >= 0 && len(s) > idx+2 && s[idx+2] == 'Z' {
				return false
			}
		}
	}
	return true
}
