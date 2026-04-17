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
	Kind               ActionKind
	Reason             string
	SaveName           string
	ForceChatWireExit  bool
	RequestID          string
	WhenEmpty          bool
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

type lifecycleManager struct {
	mu                 sync.Mutex
	hooks              LifecycleHooks
	phase              LifecyclePhase
	phaseSince         time.Time
	booted             bool
	currentPID         int
	currentGeneration  uint64
	currentAction      string
	lastError          string
	startedAt          time.Time
	readyAt            time.Time
	queue              []lifecycleRequest
	signalCh           chan struct{}
	readyCh            chan uint64
	goodbyeCh          chan uint64
	exitCh             chan processExitEvent
	started            bool
	shutdownRequested  bool
}

var (
	lifecycleMu sync.Mutex
	lifecycle   *lifecycleManager
	reqCounter  atomic.Uint64

	lifecycleStopGraceTimeout     = time.Duration(constants.MaxFactorioCloseWait) * 100 * time.Millisecond
	lifecycleStopInterruptTimeout = constants.FactorioStopInterruptTimeout
	lifecycleStopKillTimeout      = constants.FactorioStopKillTimeout
	lifecycleStopPollInterval     = 100 * time.Millisecond
	lifecyclePlayerWarnDelay      = 3 * time.Second
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
		started:    true,
	}
	lifecycle.syncCompatibilityLocked()

	go lifecycle.run()
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

	for glob.ServerRunning {
		lm.drainAsyncEvents()
		lm.checkStartupTimeout()

		if req, ok := lm.nextRequest(); ok {
			lm.execute(req)
			continue
		}

		if lm.shouldAutoStart() {
			lm.execute(lifecycleRequest{
				Request: Request{
					Kind:     ActionStart,
					Reason:   "auto-start",
					RequestID: fmt.Sprintf("auto-%d", reqCounter.Add(1)),
				},
				acceptedAt: time.Now(),
			})
			continue
		}

		select {
		case <-lm.signalCh:
		case <-ticker.C:
		}
	}
}

func (lm *lifecycleManager) shouldAutoStart() bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.phase == LifecycleStopped &&
		FactAutoStart &&
		!DoUpdateFactorio &&
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
	if req.WhenEmpty && lm.phase != LifecycleStopped && NumPlayers > 0 {
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

func (lm *lifecycleManager) execute(req lifecycleRequest) {
	started := time.Now()
	var err error

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
	if lm.hooks.WithinHours != nil && !lm.hooks.WithinHours() {
		lm.currentAction = ""
		lm.mu.Unlock()
		return errors.New("outside allowed play hours")
	}
	lm.phase = LifecycleStarting
	lm.phaseSince = time.Now()
	lm.startedAt = lm.phaseSince
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
	gen := lm.currentGeneration
	booted := lm.booted
	lm.syncCompatibilityLocked()
	lm.mu.Unlock()

	if reason == "" {
		reason = "Server quitting."
	}

	cwlog.DoLogCW("lifecycle: stop initiated generation=%d reason=%q", gen, reason)
	if FactIsRunning {
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Notice", "Quitting Factorio: "+reason, glob.COLOR_ORANGE))
	}
	glob.RelaunchThrottle = 0
	glob.NoResponseCount = 0

	if booted && NumPlayers > 0 {
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
		case <-time.After(lifecycleStopPollInterval):
		}
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
		default:
			return
		}
	}
}

func (lm *lifecycleManager) handleReadyEvent(generation uint64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if generation != lm.currentGeneration || lm.phase != LifecycleStarting {
		return
	}

	lm.phase = LifecycleRunning
	lm.phaseSince = time.Now()
	lm.booted = true
	lm.readyAt = lm.phaseSince
	lm.syncCompatibilityLocked()

	cwlog.DoLogGame("Factorio " + FactorioVersion + " is now online.")
	glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Ready", "Factorio "+FactorioVersion+" is now online.", glob.COLOR_GREEN))
	UpdateChannelName()
	DoUpdateChannelNameForce()
	glob.CrashLoopCount = 0
	cwlog.DoLogCW("lifecycle: ready observed generation=%d ready_elapsed=%v", generation, lm.readyAt.Sub(lm.startedAt).Round(time.Millisecond))
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

func (lm *lifecycleManager) finalizeStopped(generation uint64, exitErr error, report bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if generation != lm.currentGeneration && generation != 0 {
		return
	}

	wasStarting := lm.phase == LifecycleStarting
	startedAt := lm.startedAt
	stoppedAfter := time.Since(lm.phaseSince).Round(time.Millisecond)

	lm.phase = LifecycleStopped
	lm.phaseSince = time.Now()
	lm.booted = false
	lm.startedAt = time.Time{}
	lm.readyAt = time.Time{}
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
	if report {
		cwlog.DoLogCW("Factorio has closed.")
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Offline", "Factorio is now offline.", glob.COLOR_RED))
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
}

func (lm *lifecycleManager) checkStartupTimeout() {
	lm.mu.Lock()
	if lm.phase != LifecycleStarting || lm.startedAt.IsZero() || time.Since(lm.startedAt) <= constants.FactorioStartupTimeout {
		lm.mu.Unlock()
		return
	}
	lm.mu.Unlock()

	cwlog.DoLogCW("lifecycle: startup timeout hit after %v", constants.FactorioStartupTimeout)
	_ = SubmitLifecycleRequest(Request{
		Kind:     ActionRestartFactorio,
		Reason:   fmt.Sprintf("Factorio startup exceeded %v; forcing restart.", constants.FactorioStartupTimeout),
		RequestID: fmt.Sprintf("startup-timeout-%d", reqCounter.Add(1)),
	})
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
