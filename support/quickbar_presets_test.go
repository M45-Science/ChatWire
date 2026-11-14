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
		{"new", 0, "", false, 0}, {"member", 1, "", false, 2}, {"regular", 2, "", false, 4}, {"veteran", 3, "", false, 8},
		{"veteran-admin", 3, "", true, 16}, {"patron", 1, "123", false, 16},
		{"veteran-patron", 3, "123", false, 16}, {"supporter", 0, "456", false, 16},
		{"game-admin", 0, "", true, 16}, {"moderator", 255, "", true, 16}, {"supporting-admin", 255, "123", true, 16},
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
	for slot := 1; slot <= 3; slot++ {
		reply := quickbarReply{Labels: make([]string, 3)}
		if err := readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: "store", Slot: slot, Label: &label, Exchange: encoded}, &reply); err != nil || reply.Error != "" {
			t.Fatalf("store: %v %s", err, reply.Error)
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
	const exchange = "Q3:BA0VrE"
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
		if data["pending"] == true {
			w.Reset()
			FlushQuickbarPresets()
			cmd, data = capturedChatWireRequest(t, w.String())
		}
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
	if data := call("store", 8); data["error"] != nil || len(data["labels"].([]any)) != 8 {
		t.Fatal(data)
	}
	glob.PlayerList["alice"].Level = 2
	if data := call("store", 4); data["error"] != nil || len(data["labels"].([]any)) != 4 {
		t.Fatal(data)
	}
	if data := call("load", 8); data["error"] == nil || data["exchange"] != nil {
		t.Fatal(data)
	}
	glob.PlayerList["alice"].Level = 3
	if data := call("load", 8); data["exchange"] != exchange {
		t.Fatal(data)
	}
	glob.PlayerList["alice"].Level = 1
	if data := call("load", 1); data["allowed"] != true || data["exchange"] != exchange || len(data["labels"].([]any)) != 2 {
		t.Fatal(data)
	}
	if data := call("store", 2); data["error"] != nil {
		t.Fatal(data)
	}
	if data := call("store", 3); data["error"] == nil {
		t.Fatal("member could store a third preset", data)
	}
	glob.PlayerList["alice"].Level = 0
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
	if r, err := run("access", 0, 16); err != nil || r.Exchange != encoded || r.Labels[2] != "Rail" || r.Labels[15] != "" {
		t.Fatalf("legacy access: %+v %v", r, err)
	}
	if r, err := run("store", 16, 16); err != nil || r.Labels[15] != "Oil" {
		t.Fatalf("slot sixteen: %+v %v", r, err)
	}
	for _, action := range []string{"store", "load", "rename"} {
		for _, tc := range [][2]int{{3, 2}, {5, 4}, {9, 8}, {16, 2}, {17, 16}} {
			if _, err := run(action, tc[0], tc[1]); err == nil {
				t.Fatalf("%s allowed slot %d with limit %d", action, tc[0], tc[1])
			}
		}
	}
	if r, err := run("store", 2, 2); err != nil || len(r.Labels) != 2 {
		t.Fatalf("downgrade: %+v %v", r, err)
	}
	if r, err := run("load", 3, 4); err != nil || r.Exchange != encoded || r.Labels[2] != "Rail" {
		t.Fatalf("legacy slot after downgrade: %+v %v", r, err)
	}
	if r, err := run("load", 16, 16); err != nil || r.Exchange != encoded || r.Labels[15] != "Oil" {
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

func TestQuickbarExchangeEnvelopes(t *testing.T) {
	for _, exchange := range []string{"Q3:", "Q3:AAx", "Q3:_A-", "Q3:BA+", "Q3:BA", "Q3:BA!", "Q4:BA0VrE"} {
		if validQuickbarEnvelope(exchange) {
			t.Errorf("accepted malformed envelope %q", exchange)
		}
	}
	if !validQuickbarEnvelope("Q3:AA") {
		t.Fatal("empty V3 toolbar rejected")
	}
	for _, payload := range []string{"M45-QB1=rail,", "M45-QB2=AA:l,", "M45-QB4=AA,"} {
		exchange, err := fact.EncodeSoftModPayload([]byte(payload))
		if err != nil {
			t.Fatal(err)
		}
		if validQuickbarEnvelope(exchange) == strings.Contains(payload, "QB4") {
			t.Errorf("unexpected result for %q", payload)
		}
	}
}
