package support

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/disc"
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
		a.record(t0.Add(time.Duration(sec)*time.Second), who, "decon", "rail at [gps="+loc+"]", true, "")
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
		a.record(now.Add(time.Duration(i)*time.Second), "Tester", "decon", "rail at [gps=1,2]", true, "")
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

func TestEarlyMiningWeight(t *testing.T) {
	start := time.Unix(10000, 0)
	now := start.Add(5 * time.Minute)
	tests := []struct {
		name        string
		session     *activitySession
		level       int
		action      string
		delta, base int64
		want        bool
	}{
		{"new early stripping", &activitySession{joined: start, mined: 7, placed: 2}, 0, "mined", 1, 20, true},
		{"not enough removals", &activitySession{joined: start, mined: 6}, 0, "mined", 1, 20, false},
		{"building too", &activitySession{joined: start, mined: 7, placed: 3}, 0, "mined", 1, 20, false},
		{"old session", &activitySession{joined: start.Add(-11 * time.Minute), mined: 50}, 0, "mined", 1, 20, false},
		{"unobserved join", nil, 0, "mined", 1, 20, false},
		{"normal score still triggers", nil, 0, "mined", 1, 40, true},
		{"member threshold still applies", &activitySession{joined: start, mined: 50}, 1, "mined", 1, 20, false},
		{"cancellation", &activitySession{joined: start, mined: 50}, 0, "undecon", 0, 30, false},
		{"planner mark", &activitySession{joined: start, mined: 50}, 0, "decon", 2, 30, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := glob.PlayerData{Level: tc.level, SusScore: tc.base}
			got, note := tc.session.warningScore(now, tc.action, tc.delta, &p)
			if got != tc.want {
				t.Fatalf("qualified %v", got)
			}
			if tc.name == "new early stripping" && !strings.Contains(note, "5m0s ago: 8 mined, 2 placed") {
				t.Fatal(note)
			}
			if tc.name == "new early stripping" {
				if again, _ := tc.session.warningScore(now, "mined", 1, &p); again || p.SusScore != 1 || !tc.session.bonusUsed {
					t.Fatal("session boost repeated")
				}
			}
			if !got && p.SusScore != tc.base+tc.delta {
				t.Fatal("session bonus leaked into stored score", p.SusScore)
			}
		})
	}
}

func TestSessionWindowLifecycle(t *testing.T) {
	a := activityAlerts{sites: make(map[string][]*activitySite)}
	start := time.Unix(10000, 0)
	a.join("Engineer", start)
	s := a.sessions["engineer"]
	for i := 0; i < 8; i++ {
		s.observe(start, "mined")
	}
	if got := s.observe(start.Add(earlySessionWindow), "mined"); got != earlyMiningBonus {
		t.Fatal(got)
	}
	if got := s.observe(start.Add(earlySessionWindow+time.Second), "mined"); got != 0 {
		t.Fatal(got)
	}
	a.leave("ENGINEER")
	if len(a.sessions) != 0 {
		t.Fatal("leave retained session")
	}
	a.join("engineer", start)
	if a.sessions["engineer"].mined != 0 {
		t.Fatal("rejoin retained old counts")
	}
	a.due(start.Add(earlySessionWindow + time.Second))
	if len(a.sessions) != 0 {
		t.Fatal("expired session retained")
	}
}

func TestEarlyReviewAudience(t *testing.T) {
	now := time.Unix(5000, 0)
	for _, normal := range []bool{false, true} {
		a := activityAlerts{sites: make(map[string][]*activitySite)}
		a.record(now, "Tester", "mined", "rail [gps=0,0]", true, "Since joining 1m0s ago: 8 mined, 2 placed.")
		if normal {
			a.record(now, "Tester", "decon", "rail at [gps=0,1]", true, "")
		}
		got := a.due(now.Add(burstQuiet))
		if len(got) != 1 || got[0].reportOnly == normal || !strings.Contains(got[0].message, "8 mined, 2 placed") {
			t.Fatal(got)
		}
	}
}

func TestGameHeadsUpOnePerIncident(t *testing.T) {
	a := activityAlerts{sites: make(map[string][]*activitySite)}
	start := time.Unix(10000, 0)
	check := func(sec int, who, gps, review string, wantGame, wantPing bool) {
		t.Helper()
		now := start.Add(time.Duration(sec) * time.Second)
		for i := 1; i < witnessMineMinimum; i++ {
			a.record(now, who, "mined", "rail "+gps, false, "")
		}
		a.record(now, who, "mined", "rail "+gps, true, review)
		got := a.due(now.Add(burstQuiet))
		if len(got) != 1 || (got[0].gameMessage != "") != wantGame || got[0].pingRole != wantPing {
			t.Fatalf("at %d: %+v", sec, got)
		}
		if wantGame {
			for _, want := range []string{who, gps, "/report", "/banish"} {
				if !strings.Contains(got[0].gameMessage, want) {
					t.Fatal(got[0].gameMessage)
				}
			}
		}
	}
	check(0, "A", "[gps=0,0]", "early review", false, true)
	check(121, "A", "[gps=0,0]", "", true, false)
	check(362, "A", "[gps=0,0]", "", false, false)
	check(370, "B", "[gps=0,0]", "", true, true)
	check(380, "A", "[gps=800,0]", "", true, true)
	check(2300, "A", "[gps=0,0]", "", true, true)
}

func TestWitnessWarningEvidence(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		marks, cancels, mined, placed int
		gps                           string
		want                          bool
	}{
		{"net marks below", 40, 1, 0, 0, "[gps=1,2]", false},
		{"net marks at cutoff", 41, 1, 0, 0, "[gps=1,2]", true},
		{"missing location", 60, 0, 0, 0, "", false},
		{"mine below", 0, 0, 15, 0, "[gps=1,2]", false},
		{"mining and building", 0, 0, 16, 6, "[gps=1,2]", false},
		{"mine at cutoff", 0, 0, 16, 5, "[gps=1,2]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := activityAlerts{sites: make(map[string][]*activitySite)}
			now := time.Unix(1000, 0)
			for _, v := range []struct {
				action string
				n      int
			}{{"placed", tc.placed}, {"mined", tc.mined}, {"decon", tc.marks}, {"undecon", tc.cancels}} {
				for i := 0; i < v.n; i++ {
					a.record(now, "A", v.action, "rail "+tc.gps, i == 0 && (v.action == "mined" || v.action == "decon"), "")
				}
			}
			got := a.due(now.Add(burstQuiet))
			if len(got) != 1 || !got[0].pingRole || (got[0].gameMessage != "") != tc.want {
				t.Fatal(got)
			}
		})
	}
}

func TestActivityMentionOnlyOnceInReportChannel(t *testing.T) {
	oldQueue, oldReport, oldMap, oldRole := disc.CMSChan, cfg.Global.Discord.ReportChannel, cfg.Local.Channel.ChatChannel, cfg.Global.Discord.SusPingRole
	defer func() {
		disc.CMSChan, cfg.Global.Discord.ReportChannel, cfg.Local.Channel.ChatChannel, cfg.Global.Discord.SusPingRole = oldQueue, oldReport, oldMap, oldRole
	}()
	disc.CMSChan = make(chan disc.CMSBuf, 64)
	cfg.Global.Discord.ReportChannel, cfg.Local.Channel.ChatChannel, cfg.Global.Discord.SusPingRole = "report", "map", "123"
	for _, review := range []bool{true, false} {
		for _, ping := range []bool{true, false} {
			reportActivityAlert(activityAlert{message: "check", reportOnly: review, pingRole: ping})
			pings, reports, mapCopies := 0, 0, 0
			for len(disc.CMSChan) > 0 {
				m := <-disc.CMSChan
				if m.Channel == "report" {
					reports++
				} else if m.Channel == "map" {
					mapCopies++
				}
				if strings.Contains(m.Text, "<@&123>") {
					pings++
					if m.Channel != "report" {
						t.Fatal("duplicate public ping")
					}
				}
			}
			if reports == 0 || (mapCopies == 0) != review || (pings == 1) != ping || pings > 1 {
				t.Fatal(review, ping, reports, mapCopies, pings)
			}
		}
	}
}
