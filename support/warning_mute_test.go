package support

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"strings"
	"testing"
	"time"
)

func TestWarningPingMuteSharedAndExpiry(t *testing.T) {
	oldGlobal, oldLocal := cfg.Global, cfg.Local
	t.Cleanup(func() { cfg.Global, cfg.Local = oldGlobal, oldLocal })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir()
	cfg.Local.Callsign = "a"
	before := time.Now()
	until, err := SetWarningPingMute(" PlayerOne ", 30)
	if err != nil {
		t.Fatal(err)
	}
	if until.Before(before.Add(30*time.Minute)) || !warningPingMuted("PLAYERONE", before) {
		t.Fatal("mute was not applied case-insensitively")
	}
	if warningPingMuted("PlayerOne", until) {
		t.Fatal("mute did not expire")
	}
	cfg.Local.Callsign = "b"
	if !warningPingMuted("playerone", before) {
		t.Fatal("mute was not shared across maps")
	}
	if _, err := SetWarningPingMute("playerone", 0); err == nil {
		t.Fatal("invalid duration accepted")
	}
	if _, err := SetWarningPingMute(" ", 15); err == nil {
		t.Fatal("empty player accepted")
	}
}

func TestMutedWarningStillDelivered(t *testing.T) {
	oldGlobal, oldLocal, oldQueue := cfg.Global, cfg.Local, disc.CMSChan
	t.Cleanup(func() { cfg.Global, cfg.Local, disc.CMSChan = oldGlobal, oldLocal, oldQueue })
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir()
	cfg.Local.Callsign = "a"
	cfg.Global.Discord.ReportChannel, cfg.Local.Channel.ChatChannel, cfg.Global.Discord.SusPingRole = "report", "map", "123"
	cfg.Local.Options.HideActivity = false
	disc.CMSChan = make(chan disc.CMSBuf, 8)
	if _, err := SetWarningPingMute("Player", 23); err != nil {
		t.Fatal(err)
	}
	reportActivityAlert(activityAlert{player: "Player", message: "test warning", pingRole: true})
	reports, copies := 0, 0
	for len(disc.CMSChan) > 0 {
		msg := <-disc.CMSChan
		if msg.Channel == "report" {
			reports++
		}
		if msg.Channel == "map" {
			copies++
		}
		if strings.Contains(msg.Text, "<@&123>") {
			t.Fatal("muted warning pinged the role")
		}
	}
	if reports < 1 || copies < 1 {
		t.Fatalf("reports %d, map copies %d", reports, copies)
	}
	reportActivityAlert(activityAlert{player: "Another", message: "test", pingRole: true})
	found := false
	for len(disc.CMSChan) > 0 {
		msg := <-disc.CMSChan
		if strings.Contains(msg.Text, "<@&123>") && strings.Contains(msg.Text, "/mute-warning-pings") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing ping or instructions")
	}
}
