package support

import (
	"ChatWire/cfg"
	"github.com/bwmarrin/discordgo"
	"strings"
	"testing"
	"time"
)

func TestPopularMapsCalendarPeriods(t *testing.T) {
	oldGlobal, oldLocal := cfg.Global, cfg.Local
	t.Cleanup(func() { cfg.Global, cfg.Local = oldGlobal, oldLocal })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	cfg.Local.Callsign, cfg.Local.Name = "a", "Alpha"
	for _, sample := range []struct {
		day     int
		minutes int
	}{{18, 60}, {16, 120}, {10, 180}, {31, 600}} {
		d := now.AddDate(0, 0, sample.day-18)
		if sample.day == 31 {
			d = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
		}
		if err := recordMapMinutes(d, sample.minutes); err != nil {
			t.Fatal(err)
		}
	}
	cfg.Local.Callsign, cfg.Local.Name = "b", "Beta"
	if err := recordMapMinutes(now, 90); err != nil {
		t.Fatal(err)
	}
	today := popularityRanking(now, time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC))
	if len(today) != 2 || today[0].Name != "Beta" || today[0].Minutes != 90 {
		t.Fatalf("today: %+v", today)
	}
	week := popularityRanking(now, time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))
	if len(week) != 2 || week[0].Name != "Alpha" || week[0].Minutes != 180 {
		t.Fatalf("week: %+v", week)
	}
	month := popularityRanking(now, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if len(month) != 2 || month[0].Minutes != 360 {
		t.Fatalf("month: %+v", month)
	}
	report := popularMapsDescription(now)
	if !strings.Contains(report, "6.0 hours") || !strings.Contains(report, "UTC") {
		t.Fatalf("report: %s", report)
	}
}

func TestResolvePopularMapsChannel(t *testing.T) {
	calls, creates := 0, 0
	list := func(guild string) ([]*discordgo.Channel, error) {
		calls++
		return []*discordgo.Channel{{ID: "existing", Name: "popular-maps", Type: discordgo.ChannelTypeGuildText}}, nil
	}
	create := func(guild string) (*discordgo.Channel, error) {
		creates++
		return &discordgo.Channel{ID: "created"}, nil
	}
	for _, tc := range []struct{ configured, saved, want string }{{"configured", "saved", "configured"}, {"", "saved", "saved"}, {"", "", "existing"}} {
		got, err := resolvePopularMapsChannel(tc.configured, tc.saved, "guild", list, create)
		if err != nil || got != tc.want {
			t.Fatalf("got %q, %v; want %q", got, err, tc.want)
		}
	}
	if calls != 1 || creates != 0 {
		t.Fatalf("list %d, create %d", calls, creates)
	}
	empty := func(string) ([]*discordgo.Channel, error) { return nil, nil }
	got, err := resolvePopularMapsChannel("", "", "guild", empty, create)
	if err != nil || got != "created" || creates != 1 {
		t.Fatalf("creation: %q %v (%d)", got, err, creates)
	}
}
