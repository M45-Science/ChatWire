package support

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/disc"
	"github.com/bwmarrin/discordgo"
)

func TestPopularMapsRollingPeriods(t *testing.T) {
	oldGlobal, oldLocal := cfg.Global, cfg.Local
	t.Cleanup(func() { cfg.Global, cfg.Local = oldGlobal, oldLocal })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	cfg.Local.Callsign, cfg.Local.Name = "a", "Alpha"
	for _, sample := range []struct {
		age   time.Duration
		count int
	}{
		{time.Hour, 10}, {24 * time.Hour, 20}, {6 * 24 * time.Hour, 30}, {7 * 24 * time.Hour, 40}, {29 * 24 * time.Hour, 50}, {30 * 24 * time.Hour, 60},
	} {
		if err := recordMapMinutes(now.Add(-sample.age), sample.count); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		window time.Duration
		want   int
	}{{24 * time.Hour, 10}, {7 * 24 * time.Hour, 60}, {30 * 24 * time.Hour, 150}} {
		rows := popularityRanking(now, now.Add(-tc.window))
		if len(rows) != 1 || rows[0].Minutes != tc.want {
			t.Fatalf("window %v: %+v, want %d", tc.window, rows, tc.want)
		}
	}
	report := popularMapsDescription(now)
	for _, want := range []string{"**Last 24 hours**", "**Last 7 days**", "**Last 30 days**"} {
		if !strings.Contains(report, want) {
			t.Fatalf("missing %q: %s", want, report)
		}
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

type popularityTransport func(*http.Request) (*http.Response, error)

func (f popularityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublishPopularMapsEditsOnlyWhenDisplayedContentChanges(t *testing.T) {
	oldGlobal, oldLocal, oldSession := cfg.Global, cfg.Local, disc.DS
	t.Cleanup(func() { cfg.Global, cfg.Local, disc.DS = oldGlobal, oldLocal, oldSession })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir()
	cfg.Global.Discord.PopularMapsChannel = "123"
	cfg.Local.Callsign, cfg.Local.Name = "a", "Alpha"
	session, err := discordgo.New("Bot test")
	if err != nil {
		t.Fatal(err)
	}
	disc.DS = session
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

	var posted *discordgo.Message
	var methods []string
	session.Client = &http.Client{Transport: popularityTransport(func(r *http.Request) (*http.Response, error) {
		methods = append(methods, r.Method)
		status := http.StatusOK
		var result any
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			var payload discordgo.Message
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Embeds) != 1 {
				t.Fatalf("expected one embed in %s", r.Method)
			}
			payload.ID = "456"
			payload.Embeds[0].Type = discordgo.EmbedTypeRich
			posted = &payload
			result = posted
		case http.MethodGet:
			result = posted
			if posted == nil {
				status = http.StatusNotFound
				result = discordgo.APIErrorMessage{Code: discordgo.ErrCodeUnknownMessage}
			}
		default:
			t.Fatalf("unexpected Discord request: %s", r.Method)
		}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(data)), Request: r}, nil
	})}

	refresh := func(want string) {
		t.Helper()
		methods = nil
		if err := publishPopularMaps(now); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(methods, " "); got != want {
			t.Fatalf("requests = %q, want %q", got, want)
		}
	}
	if err := recordMapMinutes(now, 60); err != nil {
		t.Fatal(err)
	}
	refresh("POST")
	now = now.Add(15 * time.Minute)
	if err := recordMapMinutes(now, 1); err != nil {
		t.Fatal(err)
	} // Still displayed as 1.0 hours.
	refresh("GET")
	if err := recordMapMinutes(now, 60); err != nil {
		t.Fatal(err)
	}
	refresh("GET PATCH")
	refresh("GET")
	posted = nil // Deleted from Discord, while the saved message ID is still present.
	refresh("GET POST")
}
