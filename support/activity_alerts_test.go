package support

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/glob"
)

func TestDeconstructionProtocolBurst(t *testing.T) {
	resetSupportTestState(t)
	// Run the same decoded activity/message dispatch used by the SoftMod protocol.
	p := &glob.PlayerData{Name: "Tester"}
	glob.PlayerList["tester"] = p
	oldMute := cfg.Global.Options.ShutupSusWarn
	cfg.Global.Options.ShutupSusWarn = false
	t.Cleanup(func() { cfg.Global.Options.ShutupSusWarn = oldMute; clear(mapActivityAlerts.sites) })
	clear(mapActivityAlerts.sites)
	send := func(action string) { dispatchChatWireTextEvent("activity", "Tester "+action) }
	for i := 0; i < 294; i++ {
		send(fmt.Sprintf("decon %s at [gps=%d,-5,space platform]", []string{"belt", "inserter", "pole"}[i%3], i%16))
	}
	score := p.SusScore
	send("decon [gps=0,-5,space platform] 44m²")
	dispatchChatWireTextEvent("message", "[ACT] Tester decon belt at [gps=1,-5,space platform]")
	for i := 0; i < 10; i++ {
		send("undecon belt at [gps=1,-5,space platform]")
	}
	if p.SusScore != score {
		t.Fatalf("score changed on duplicate, area or cancellation: %d -> %d", score, p.SusScore)
	}
	now := time.Now()
	if got := mapActivityAlerts.due(now.Add(time.Second)); len(got) != 0 {
		t.Fatal(got)
	}
	got := mapActivityAlerts.due(now.Add(4 * time.Second))
	if len(got) != 1 {
		t.Fatal(got)
	}
	for _, want := range []string{"Tester", "[gps=0,-5,space platform]", "294 marks", "98 × belt", "98 × inserter", "98 × pole", "10 cancellation events"} {
		if !strings.Contains(got[0].message, want) {
			t.Errorf("missing %s: %s", want, got[0].message)
		}
	}
	p.SusScore = 100
	send("undecon belt at [gps=1,1]")
	if p.SusScore != 100 || len(mapActivityAlerts.due(now.Add(20*time.Minute))) != 0 {
		t.Fatal("cancel triggered a new incident")
	}
}

func TestActivitySitesBackOffIndependently(t *testing.T) {
	a := activityAlerts{sites: make(map[string][]*activitySite)}
	t0 := time.Unix(1000, 0)
	mark := func(sec int, who, loc string) {
		a.record(t0.Add(time.Duration(sec)*time.Second), who, "decon", "rail at [gps="+loc+"]", true)
	}
	due := func(sec, want int) {
		t.Helper()
		got := a.due(t0.Add(time.Duration(sec) * time.Second))
		if len(got) != want {
			t.Fatalf("at %d: %v, want %d", sec, got, want)
		}
	}
	mark(0, "Alice", "-1,0")
	mark(0, "alice", "1,0,nauvis")
	due(3, 1)
	mark(4, "Alice", "10,0")
	mark(4, "Bob", "10,0")
	mark(4, "Alice", "600,0")
	mark(4, "Alice", "10,0,gleba")
	due(7, 3)
	due(122, 0)
	due(123, 1)
	mark(124, "Alice", "0,0")
	due(362, 0)
	due(363, 1)
	mark(2200, "Alice", "0,0")
	due(2203, 1)
	due(4100, 0)
	if len(a.sites) != 0 {
		t.Fatal("stale sites retained")
	}
}

func TestActivityContinuousBurstAndRankThreshold(t *testing.T) {
	a := activityAlerts{sites: make(map[string][]*activitySite)}
	now := time.Unix(1000, 0)
	for i := 0; i <= 15; i++ {
		a.record(now.Add(time.Duration(i)*time.Second), "Tester", "decon", "rail at [gps=1,2]", true)
	}
	if reports := a.due(now.Add(burstLimit)); len(reports) != 1 || !strings.Contains(reports[0].message, "16 marks") {
		t.Fatal(reports)
	}
	resetSupportTestState(t)
	clear(mapActivityAlerts.sites)
	t.Cleanup(func() { clear(mapActivityAlerts.sites) })
	p := &glob.PlayerData{Name: "Tester", Level: 1}
	glob.PlayerList["tester"] = p
	for i := 0; i < 21; i++ {
		scoreActivity("Tester", "decon", "rail at [gps=1,2]")
	}
	if p.SusScore != 42 || len(mapActivityAlerts.due(time.Now().Add(burstLimit))) != 0 {
		t.Fatal("member threshold lost")
	}
	p.Level = 2
	scoreActivity("Tester", "decon", "rail at [gps=1,2]")
	if p.SusScore != 0 {
		t.Fatal("regular was scored")
	}
}
