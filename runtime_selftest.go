package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/support"
)

var defaultRuntimeSelfTestCases = []string{
	"start",
	"restart",
	"stop",
	"change-map",
	"update-check",
	"chatwire-reboot",
}

func runRuntimeSelfTest(casesCSV string, stepTimeout time.Duration, chatwireExitCh <-chan bool) error {
	if stepTimeout <= 0 {
		stepTimeout = 5 * time.Minute
	}

	cases := parseRuntimeSelfTestCases(casesCSV)
	if len(cases) == 0 {
		cases = append([]string(nil), defaultRuntimeSelfTestCases...)
	}

	if err := ensureLifecycleStopped(stepTimeout); err != nil {
		return fmt.Errorf("initial stop precondition failed: %w", err)
	}

	for _, testCase := range cases {
		cwlog.DoLogCW("runtimeSelfTest: starting case=%s timeout=%v", testCase, stepTimeout)
		started := time.Now()

		var err error
		switch testCase {
		case "start":
			err = runStartCase(stepTimeout)
		case "stop":
			err = runStopCase(stepTimeout)
		case "restart":
			err = runRestartCase(stepTimeout)
		case "change-map":
			err = runChangeMapCase(stepTimeout)
		case "map-reset":
			err = runMapResetCase(stepTimeout)
		case "update-check":
			err = runUpdateCheckCase()
		case "update-install":
			err = runUpdateInstallCase()
		case "mod-update-check":
			err = runModUpdateCheckCase()
		case "sync-mods":
			err = runSyncModsCase()
		case "chatwire-reboot":
			err = runChatWireRebootCase(stepTimeout, chatwireExitCh)
		default:
			err = fmt.Errorf("unknown runtime self-test case: %s", testCase)
		}
		if err != nil {
			return fmt.Errorf("case %q failed after %v: %w", testCase, time.Since(started).Round(time.Millisecond), err)
		}
		fact.SetAutolaunch(false, false)

		cwlog.DoLogCW("runtimeSelfTest: passed case=%s elapsed=%v", testCase, time.Since(started).Round(time.Millisecond))
	}

	return nil
}

func parseRuntimeSelfTestCases(casesCSV string) []string {
	casesCSV = strings.TrimSpace(casesCSV)
	if casesCSV == "" || strings.EqualFold(casesCSV, "default") {
		return append([]string(nil), defaultRuntimeSelfTestCases...)
	}

	parts := strings.Split(casesCSV, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func runStartCase(timeout time.Duration) error {
	if err := fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionStart, Reason: "runtime self-test start"}); err != nil {
		return err
	}
	return waitForLifecycleRunning(timeout)
}

func runStopCase(timeout time.Duration) error {
	if err := fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionStop, Reason: "runtime self-test stop"}); err != nil {
		return err
	}
	return ensureLifecycleStopped(timeout)
}

func runRestartCase(timeout time.Duration) error {
	before := fact.GetLifecycleState()
	if err := fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionRestartFactorio, Reason: "runtime self-test restart"}); err != nil {
		return err
	}
	return waitForLifecycleRestart(timeout, before)
}

func runChangeMapCase(timeout time.Duration) error {
	saveName, err := chooseRuntimeTestSave()
	if err != nil {
		return err
	}
	if err := fact.SubmitLifecycleRequest(fact.Request{
		Kind:     fact.ActionChangeMap,
		Reason:   "runtime self-test change-map",
		SaveName: saveName,
	}); err != nil {
		return err
	}
	return waitForLifecycleRunning(timeout)
}

func runMapResetCase(timeout time.Duration) error {
	if err := fact.SubmitLifecycleRequest(fact.Request{
		Kind:   fact.ActionMapReset,
		Reason: "runtime self-test map-reset",
	}); err != nil {
		return err
	}
	return waitForLifecycleRunning(timeout)
}

func runUpdateCheckCase() error {
	_, msg, err, _ := factUpdater.DoQuickLatest(false)
	if err {
		return fmt.Errorf("update-check failed: %s", msg)
	}
	cwlog.DoLogCW("runtimeSelfTest: update-check result=%s", msg)
	return nil
}

func runUpdateInstallCase() error {
	if err := fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionStop, Reason: "runtime self-test update-install"}); err != nil {
		return err
	}
	if err := ensureLifecycleStopped(2 * time.Minute); err != nil {
		return err
	}
	_, msg, err, _ := factUpdater.DoQuickLatest(true)
	if err {
		return fmt.Errorf("update-install failed: %s", msg)
	}
	cwlog.DoLogCW("runtimeSelfTest: update-install result=%s", msg)
	return nil
}

func runModUpdateCheckCase() error {
	glob.UpdatersLock.Lock()
	defer glob.UpdatersLock.Unlock()

	updated, err := modupdate.CheckModUpdates(true, false, false)
	if err != nil && err.Error() != "no mod updates available" && err.Error() != "the game has no installed mods to update" {
		return fmt.Errorf("mod-update-check failed: %v", err)
	}
	cwlog.DoLogCW("runtimeSelfTest: mod-update-check updated=%t err=%v", updated, err)
	return nil
}

func runSyncModsCase() error {
	if err := fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionStop, Reason: "runtime self-test sync-mods"}); err != nil {
		return err
	}
	if err := ensureLifecycleStopped(2 * time.Minute); err != nil {
		return err
	}
	if !support.SyncMods(nil, "") {
		return fmt.Errorf("sync-mods failed")
	}
	return nil
}

func runChatWireRebootCase(timeout time.Duration, chatwireExitCh <-chan bool) error {
	fact.SetAutolaunch(false, false)
	drainRuntimeSelfTestExitChannel(chatwireExitCh)
	if err := fact.SubmitLifecycleRequest(fact.Request{
		Kind:   fact.ActionRestartChatWire,
		Reason: "runtime self-test chatwire-reboot",
	}); err != nil {
		return err
	}
	if err := ensureLifecycleStopped(timeout); err != nil {
		return err
	}
	select {
	case <-chatwireExitCh:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for chatwire exit hook")
	}
}

func chooseRuntimeTestSave() (string, error) {
	found, fileName, _ := support.GetSaveGame(true)
	if !found || fileName == "" {
		return "", fmt.Errorf("no valid save found for change-map test")
	}
	base := filepath.Base(fileName)
	if !strings.HasSuffix(strings.ToLower(base), ".zip") {
		return "", fmt.Errorf("selected save is not a zip: %s", base)
	}
	return strings.TrimSuffix(base, filepath.Ext(base)), nil
}

func waitForLifecycleRunning(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		state := fact.GetLifecycleState()
		if state.Phase == fact.LifecycleRunning && state.Booted {
			return nil
		}
		if state.Phase == fact.LifecycleStopped && state.LastError != "" {
			return fmt.Errorf("lifecycle stopped with error: %s", state.LastError)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for lifecycle running; phase=%s booted=%t last_error=%q", state.Phase, state.Booted, state.LastError)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func waitForLifecycleRestart(timeout time.Duration, before fact.State) error {
	return waitForLifecycleRestartWithGetter(timeout, before, fact.GetLifecycleState)
}

func waitForLifecycleRestartWithGetter(timeout time.Duration, before fact.State, getState func() fact.State) error {
	deadline := time.Now().Add(timeout)
	sawNotRunning := before.Phase == fact.LifecycleStopped
	for {
		state := getState()
		if state.Phase != fact.LifecycleRunning || !state.Booted {
			sawNotRunning = true
		}
		if sawNotRunning &&
			state.Phase == fact.LifecycleRunning &&
			state.Booted &&
			(state.Since.After(before.Since) || state.PID != before.PID) {
			return nil
		}
		if state.Phase == fact.LifecycleStopped && state.LastError != "" {
			return fmt.Errorf("lifecycle stopped with error: %s", state.LastError)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for lifecycle restart; phase=%s booted=%t pid=%d last_error=%q", state.Phase, state.Booted, state.PID, state.LastError)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func ensureLifecycleStopped(timeout time.Duration) error {
	if ok := fact.WaitForLifecycleStop(timeout); ok {
		return nil
	}
	state := fact.GetLifecycleState()
	return fmt.Errorf("timed out waiting for lifecycle stop; phase=%s booted=%t last_error=%q", state.Phase, state.Booted, state.LastError)
}

func drainRuntimeSelfTestExitChannel(ch <-chan bool) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
