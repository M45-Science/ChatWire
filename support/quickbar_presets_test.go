package support

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"github.com/bwmarrin/discordgo"
)

func TestQuickbarRoleAccess(t *testing.T) {
	resetSupportTestState(t)
	oldGuild, oldSession, oldRoles := disc.Guild, disc.DS, cfg.Global.Discord.Roles.RoleCache
	t.Cleanup(func() { disc.Guild, disc.DS, cfg.Global.Discord.Roles.RoleCache = oldGuild, oldSession, oldRoles })
	disc.DS = &discordgo.Session{}
	disc.Guild = &discordgo.Guild{Members: []*discordgo.Member{{User: &discordgo.User{ID: "123"}, Roles: []string{"paid"}}}}
	disc.Guild.Members = append(disc.Guild.Members, &discordgo.Member{User: &discordgo.User{ID: "456"}, Roles: []string{"supporter"}})
	cfg.Global.Discord.Roles.RoleCache.Supporter = "supporter"
	cfg.Global.Discord.Roles.RoleCache.Patreon = "paid"
	for _, tc := range []struct {
		name  string
		level int
		role  string
		admin bool
		want  int
	}{
		{"member", 1, "", false, 0}, {"regular", 2, "", false, 3}, {"veteran", 3, "", false, 5}, {"veteran-admin", 3, "", true, 10}, {"patron", 1, "123", false, 10},
		{"veteran-patron", 3, "123", false, 10}, {"supporter", 0, "456", false, 10},
		{"game-admin", 0, "", true, 10}, {"moderator", 255, "", true, 10}, {"supporting-admin", 255, "123", true, 10},
		{"demoted-admin", 255, "", false, 0}, {"banned", -1, "123", true, 0},
	} {
		glob.PlayerList[tc.name] = &glob.PlayerData{Level: tc.level, ID: tc.role}
		if got := quickbarSlotLimit(quickbarRequest{Name: strings.ToUpper(tc.name), Admin: tc.admin}); got != tc.want {
			t.Errorf("%s: got %v", tc.name, got)
		}
	}
}

func TestQuickbarIndependentSlotsAndPlayers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quickbar-presets.json")
	encoded, err := fact.EncodeSoftModPayload([]byte("M45-QB2=iron-plate,"))
	if err != nil {
		t.Fatal(err)
	}
	label := "Rails"
	done := make(chan error, 3)
	for slot := 1; slot <= 3; slot++ {
		go func(slot int) {
			done <- readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: "store", Slot: slot, Label: &label, Exchange: encoded}, &quickbarReply{Labels: make([]string, 3)})
		}(slot)
	}
	for range 3 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	for slot := 1; slot <= 3; slot++ {
		result := quickbarReply{Labels: make([]string, 3)}
		if err := readWriteQuickbar(path, quickbarRequest{Name: "ALICE", Action: "load", Slot: slot}, &result); err != nil || result.Exchange != encoded || !slices.Equal(result.Labels, []string{"", "Rails", "Rails"}) {
			t.Fatalf("slot %d: %#v %v", slot, result, err)
		}
	}
	for _, tc := range []struct {
		name  string
		first bool
		want  string
	}{{"Alice", true, encoded}, {"Alice", false, ""}, {"Bob", true, ""}} {
		result := quickbarReply{Labels: make([]string, 3)}
		if err := readWriteQuickbar(path, quickbarRequest{Name: tc.name, Action: "access", FirstJoin: tc.first}, &result); err != nil || result.Exchange != tc.want {
			t.Fatalf("access: %#v %v", result, err)
		}
	}
	if err := os.WriteFile(path, []byte("corrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: "store", Slot: 1, Exchange: encoded}, &quickbarReply{Labels: make([]string, 3)}); err == nil {
		t.Fatal("overwrote corrupt database")
	}
}

func TestQuickbarWireRoundTrip(t *testing.T) {
	resetSupportTestState(t)
	oldRoot := cfg.Global.Paths.Folders.ServersRoot
	cfg.Global.Paths.Folders.ServersRoot = t.TempDir()
	w := &testWriteCloser{}
	fact.SetFactorioPipe(w, 0)
	t.Cleanup(func() { cfg.Global.Paths.Folders.ServersRoot = oldRoot; fact.SetFactorioPipe(nil, 0) })
	exchange, err := fact.EncodeSoftModPayload([]byte("M45-QB2=iron-plate,"))
	if err != nil {
		t.Fatal(err)
	}
	call := func(action string, slot int) map[string]any {
		t.Helper()
		w.Reset()
		line, err := json.Marshal(map[string]any{"v": 1, "kind": "event", "event": "quickbar-preset", "data": map[string]any{"name": "Alice", "id": 8, "action": action, "slot": slot, "exchange": exchange, "first_join": true}})
		if err != nil {
			t.Fatal(err)
		}
		if !handleChatWire(chatWireTestLine(t, string(line))) {
			t.Fatal("event not handled")
		}
		cmd, data := capturedChatWireRequest(t, w.String())
		if cmd != "quickbar-result" || data["id"] != float64(8) {
			t.Fatalf("unexpected reply: %s %#v", cmd, data)
		}
		return data
	}
	if data := call("store", 1); data["allowed"] != false {
		t.Fatal(data)
	}
	glob.PlayerList["alice"] = &glob.PlayerData{Level: 3}
	if data := call("store", 1); data["error"] != nil || data["allowed"] != true {
		t.Fatal(data)
	}
	if data := call("access", 1); data["exchange"] != exchange {
		t.Fatal(data)
	}
	if data := call("store", 5); data["error"] != nil || len(data["labels"].([]any)) != 5 {
		t.Fatal(data)
	}
	glob.PlayerList["alice"].Level = 2
	if data := call("store", 3); data["error"] != nil || len(data["labels"].([]any)) != 3 {
		t.Fatal(data)
	}
	if data := call("load", 5); data["error"] == nil || data["exchange"] != nil {
		t.Fatal(data)
	}
	glob.PlayerList["alice"].Level = 3
	if data := call("load", 5); data["exchange"] != exchange {
		t.Fatal(data)
	}
	glob.PlayerList["alice"].Level = 1
	if data := call("load", 1); data["allowed"] != false || data["exchange"] != nil {
		t.Fatal(data)
	}
}

func TestQuickbarLegacySlotsAndRoleChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quickbar-presets.json")
	encoded, err := fact.EncodeSoftModPayload([]byte("M45-QB2=rail,"))
	if err != nil {
		t.Fatal(err)
	}
	legacy := map[string][3]quickbarPreset{"alice": {{Exchange: encoded}, {}, {Label: "Rail", Exchange: encoded}}}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	label := "Oil"
	run := func(action string, slot, limit int) (quickbarReply, error) {
		r := quickbarReply{Labels: make([]string, limit)}
		err := readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: action, Slot: slot, FirstJoin: true, Label: &label, Exchange: encoded}, &r)
		return r, err
	}
	if r, err := run("access", 0, 10); err != nil || r.Exchange != encoded || r.Labels[2] != "Rail" || r.Labels[9] != "" {
		t.Fatalf("legacy access: %+v %v", r, err)
	}
	if r, err := run("store", 10, 10); err != nil || r.Labels[9] != "Oil" {
		t.Fatalf("slot ten: %+v %v", r, err)
	}
	for _, action := range []string{"store", "load", "rename"} {
		for _, tc := range [][2]int{{4, 3}, {6, 5}, {10, 3}, {11, 10}} {
			if _, err := run(action, tc[0], tc[1]); err == nil {
				t.Fatalf("%s allowed slot %d with limit %d", action, tc[0], tc[1])
			}
		}
	}
	if r, err := run("store", 2, 3); err != nil || len(r.Labels) != 3 || r.Labels[2] != "Rail" {
		t.Fatalf("downgrade: %+v %v", r, err)
	}
	if r, err := run("load", 10, 10); err != nil || r.Exchange != encoded || r.Labels[9] != "Oil" {
		t.Fatalf("restored slot: %+v %v", r, err)
	}
}

func TestQuickbarRenamePreservesLayoutAndDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quickbar-presets.json")
	exchange, err := fact.EncodeSoftModPayload([]byte("M45-QB2=rail,"))
	if err != nil {
		t.Fatal(err)
	}
	slot := 1
	call := func(action string, label *string) quickbarReply {
		t.Helper()
		reply := quickbarReply{Labels: make([]string, 3)}
		req := quickbarRequest{Name: "Alice", Action: action, Slot: slot, Label: label, FirstJoin: true}
		if action == "store" {
			req.Exchange = exchange
		}
		if err := readWriteQuickbar(path, req, &reply); err != nil {
			t.Fatal(err)
		}
		return reply
	}
	name := "Rail depot"
	call("store", &name)
	if r := call("rename", &name); r.Error == "" {
		t.Fatal("renamed Default")
	}
	if r := call("access", nil); r.Labels[0] != "" || r.Exchange != exchange {
		t.Fatal(r)
	}
	slot = 2
	if r := call("rename", &name); r.Labels[1] != name {
		t.Fatal(r)
	}
	if r := call("load", nil); r.Exchange != "" {
		t.Fatal("renaming empty slot created a layout")
	}
	call("store", nil)
	name = "Oil"
	call("rename", &name)
	if r := call("access", nil); r.Exchange != exchange || r.Labels[1] != name {
		t.Fatal(r)
	}
	tooLong := strings.Repeat("x", 33)
	if r := call("rename", &tooLong); r.Error == "" {
		t.Fatal("accepted oversized name")
	}
	name = ""
	call("rename", &name)
	if r := call("load", nil); r.Exchange != exchange || r.Labels[1] != "" {
		t.Fatal(r)
	}
}
