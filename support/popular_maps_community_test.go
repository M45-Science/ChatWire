package support

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/glob"
)

func TestPopularMapsCommunityEmbed(t *testing.T) {
	root := t.TempDir()
	oldGlobal, oldPlayers := cfg.Global, glob.PlayerList
	t.Cleanup(func() { cfg.Global, glob.PlayerList = oldGlobal, oldPlayers })
	cfg.Global.Paths.Folders.ServersRoot, cfg.Global.Paths.ChatWirePrefix = root, "instance-"
	glob.PlayerList = map[string]*glob.PlayerData{
		"new": {}, "member": {Level: 1}, "regular": {Level: 2},
		"vet1": {Level: 3}, "vet2": {Level: 3}, "mod": {Level: 255},
		"banned": {Level: -1}, "deleted": {Level: -255},
	}

	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	installMap := func(name string, channel string) string {
		t.Helper()
		dir := filepath.Join(root, "instance-"+name)
		if err := os.MkdirAll(filepath.Join(dir, "audit-log"), 0755); err != nil {
			t.Fatal(err)
		}
		config := fmt.Sprintf(`{"Name":%q,"Channel":{"ChatChannel":%q}}`, name, channel)
		if err := os.WriteFile(filepath.Join(dir, constants.CWLocalConfig), []byte(config), 0600); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	event := func(dir string, at time.Time, message string) {
		t.Helper()
		path := filepath.Join(dir, "audit-log", at.UTC().Format("cw-2-January-2006.log"))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		local := at.Local()
		_, err = fmt.Fprintf(f, "%2d:%2d.%2d: %15s:%5d: lifecycle: %s\n", local.Hour(), local.Minute(), local.Second(), "lifecycle.go", 804, message)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	a := installMap("Railworld", "100")
	b := installMap("Peaceful", "")
	c := installMap("Space-Age", "")
	d := installMap("Earlier", "")
	completed := "follow-up action executed kind=map-reset elapsed=4s"
	mostRecent, second := now.Add(-2*time.Hour), now.Add(-72*time.Hour)
	event(a, now.Add(-24*time.Hour), completed)
	event(a, mostRecent, completed)
	event(b, second, completed)
	third := now.AddDate(0, 0, -12)
	event(c, third, completed)
	event(d, third.Add(-time.Hour), completed)
	event(c, now, "request accepted kind=map-reset")
	event(c, now, "follow-up action executed kind=restart-chatwire")
	event(c, now.Add(time.Hour), completed)

	embed := popularMapsEmbed(now)
	for _, want := range []string{"**Last 24 hours**", "**Last 7 days**", "**Last 30 days**", "[m45sci.xyz](https://m45sci.xyz)"} {
		if !strings.Contains(embed.Description, want) {
			t.Errorf("missing %q in description", want)
		}
	}
	if got, want := embed.Description, "Members: 1 · Regulars: 1 · Veterans: 2\n"; !strings.HasPrefix(got, want) {
		t.Errorf("description %q should start with %q", got, want)
	}
	want := fmt.Sprintf("\n**Recently reset**\n1. <#100> — <t:%d:R>\n2. Peaceful — <t:%d:R>\n3. Space-Age — <t:%d:R>", mostRecent.Unix(), second.Unix(), third.Unix())
	if got := embed.Description; !strings.HasSuffix(got, want) {
		t.Errorf("recent resets = %q, want %q", got, want)
	}
	if got := recentResetSummary(third.AddDate(0, 0, -1)); got != "No resets recorded." {
		t.Errorf("expected empty history before the first reset, got %q", got)
	}
}
