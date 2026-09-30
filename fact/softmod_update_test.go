package fact

import (
	"ChatWire/cfg"
	"ChatWire/glob"
	"errors"
	"testing"
	"time"
)

func setupSoftModUpdateTest(t *testing.T, hooks LifecycleHooks) *lifecycleManager {
	t.Helper()
	resetLifecycleTestState(t)
	oldLocal, oldVersion := cfg.Local, FactorioVersion
	oldCrashes, oldLastCrash := glob.CrashLoopCount, glob.LastCrash
	oldPending := factorioUpdatePending.Swap(false)
	t.Cleanup(func() {
		lifecycleMu.Lock()
		lifecycle = nil
		lifecycleMu.Unlock()
		cfg.Local = oldLocal
		SetFactorioVersion(oldVersion)
		glob.CrashLoopCount, glob.LastCrash = oldCrashes, oldLastCrash
		factorioUpdatePending.Store(oldPending)
	})
	cfg.Local.Options.SoftModOptions.InjectSoftMod = true
	glob.CrashLoopCount = 0
	cfg.Local.PendingSave = ""
	cfg.Local.Settings.NewMap = false
	cfg.Local.Channel.ChatChannel = ""
	SetFactorioVersion("2.0.76")
	lm := newTestLifecycleManager(hooks)
	lifecycleMu.Lock()
	lifecycle = lm
	lifecycleMu.Unlock()
	return lm
}

func TestFactorioUpdateRestartsOnceWhenSoftModMissing(t *testing.T) {
	launches := 0
	lm := setupSoftModUpdateTest(t, LifecycleHooks{LaunchFactorio: func(uint64, string) error { launches++; return nil }})
	MarkFactorioUpdated()
	if err := lm.executeStart("auto-start after update", ""); err != nil {
		t.Fatal(err)
	}
	first := lm.currentGeneration
	lm.handleReadyEvent(first)
	lm.checkSoftModAfterUpdate(lm.readyAt.Add(softModUpdateStartupGrace - time.Nanosecond))
	if len(lm.queue) != 0 {
		t.Fatal("reboot queued before handshake grace elapsed")
	}
	lm.checkSoftModAfterUpdate(lm.readyAt.Add(softModUpdateStartupGrace))
	lm.checkSoftModAfterUpdate(lm.readyAt.Add(2 * softModUpdateStartupGrace))
	if len(lm.queue) != 1 || lm.queue[0].Kind != ActionRestartFactorio {
		t.Fatalf("recovery queue=%+v", lm.queue)
	}
	req, ok := lm.nextRequest()
	if !ok {
		t.Fatal("recovery restart not runnable")
	}
	lm.execute(req)
	if launches != 2 || lm.currentGeneration != first+1 {
		t.Fatalf("launches=%d generation=%d", launches, lm.currentGeneration)
	}
	lm.handleReadyEvent(lm.currentGeneration)
	lm.checkSoftModAfterUpdate(lm.readyAt.Add(time.Hour))
	if len(lm.queue) != 0 || lm.softModUpdateRecovery {
		t.Fatal("recovery reboot armed another reboot loop")
	}
}

func TestFactorioUpdateSoftModRecoveryConditions(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		updated, enabled, detected, staleHello, ready bool
		want                                          bool
	}{
		{name: "ordinary restart", enabled: true, ready: true},
		{name: "SoftMod disabled", updated: true, ready: true},
		{name: "SoftMod detected", updated: true, enabled: true, detected: true, ready: true},
		{name: "startup incomplete", updated: true, enabled: true},
		{name: "stale hello", updated: true, enabled: true, staleHello: true, ready: true, want: true},
		{name: "missing SoftMod", updated: true, enabled: true, ready: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lm := setupSoftModUpdateTest(t, LifecycleHooks{LaunchFactorio: func(uint64, string) error { return nil }})
			cfg.Local.Options.SoftModOptions.InjectSoftMod = tc.enabled
			if tc.updated {
				MarkFactorioUpdated()
			}
			if err := lm.executeStart("test", ""); err != nil {
				t.Fatal(err)
			}
			if tc.detected {
				NotifySoftModDetected(lm.currentGeneration)
			}
			if tc.staleHello {
				NotifySoftModDetected(lm.currentGeneration + 1)
			}
			if tc.ready {
				lm.handleReadyEvent(lm.currentGeneration)
			}
			lm.checkSoftModAfterUpdate(time.Now().Add(time.Hour))
			if got := len(lm.queue) > 0; got != tc.want {
				t.Fatalf("queued recovery=%t want=%t", got, tc.want)
			}
		})
	}
}

func TestFactorioUpdateRecoverySurvivesFailedLaunch(t *testing.T) {
	failed := true
	lm := setupSoftModUpdateTest(t, LifecycleHooks{LaunchFactorio: func(uint64, string) error {
		if failed {
			return errors.New("launch failed")
		}
		return nil
	}})
	MarkFactorioUpdated()
	if err := lm.executeStart("test", ""); err == nil {
		t.Fatal("expected failed launch")
	}
	if !factorioUpdatePending.Load() {
		t.Fatal("failed launch consumed update verification")
	}
	failed = false
	if err := lm.executeStart("retry", ""); err != nil {
		t.Fatal(err)
	}
	lm.handleReadyEvent(lm.currentGeneration)
	lm.checkSoftModAfterUpdate(lm.readyAt.Add(softModUpdateStartupGrace))
	if len(lm.queue) != 1 {
		t.Fatal("successful retry did not verify SoftMod")
	}
}

func TestFactorioUpdateRecoveryDefersToPendingLifecycleAction(t *testing.T) {
	lm := setupSoftModUpdateTest(t, LifecycleHooks{LaunchFactorio: func(uint64, string) error { return nil }})
	MarkFactorioUpdated()
	if err := lm.executeStart("test", ""); err != nil {
		t.Fatal(err)
	}
	lm.handleReadyEvent(lm.currentGeneration)
	lm.queue = append(lm.queue, lifecycleRequest{Request: Request{Kind: ActionStop}})
	lm.checkSoftModAfterUpdate(lm.readyAt.Add(softModUpdateStartupGrace))
	if len(lm.queue) != 1 || lm.queue[0].Kind != ActionStop {
		t.Fatalf("recovery overrode queued stop: %+v", lm.queue)
	}
}

func TestFactorioUpdateRecoverySurvivesInterruptedStartup(t *testing.T) {
	lm := setupSoftModUpdateTest(t, LifecycleHooks{LaunchFactorio: func(uint64, string) error { return nil }})
	MarkFactorioUpdated()
	if err := lm.executeStart("test", ""); err != nil {
		t.Fatal(err)
	}
	lm.finalizeStopped(lm.currentGeneration, errors.New("startup failed"), false)
	if !factorioUpdatePending.Load() {
		t.Fatal("startup failure lost update verification")
	}
	if err := lm.executeStart("retry", ""); err != nil {
		t.Fatal(err)
	}
	lm.handleReadyEvent(lm.currentGeneration)
	lm.checkSoftModAfterUpdate(lm.readyAt.Add(softModUpdateStartupGrace))
	if len(lm.queue) != 1 {
		t.Fatal("retry did not verify missing SoftMod")
	}
}
