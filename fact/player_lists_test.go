package fact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/glob"
)

func configurePlayerListFiles(t *testing.T) string {
	t.Helper()

	oldGlobal := cfg.Global
	oldLocal := cfg.Local
	oldPlayers := glob.PlayerList
	t.Cleanup(func() {
		cfg.Global = oldGlobal
		cfg.Local = oldLocal
		glob.PlayerList = oldPlayers
	})

	root := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = root + string(os.PathSeparator)
	cfg.Global.Paths.ChatWirePrefix = "cw-"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Local.Callsign = "test"
	if err := os.MkdirAll(cfg.GetFactorioFolder(), 0755); err != nil {
		t.Fatal(err)
	}
	return cfg.GetFactorioFolder()
}

func readPlayerNames(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		t.Fatalf("invalid player list JSON %q: %v", data, err)
	}
	return names
}

func TestWriteAdminlistSinglePlayerIsValidJSON(t *testing.T) {
	factorioDir := configurePlayerListFiles(t)
	glob.PlayerList = map[string]*glob.PlayerData{
		"admin":     {Name: "admin", Level: 255},
		"admin-254": {Name: "admin-254", Level: 254},
	}

	if got := WriteAdminlist(); got != 2 {
		t.Fatalf("WriteAdminlist() = %d, want 2", got)
	}
	want := []string{"admin", "admin-254"}
	got := readPlayerNames(t, filepath.Join(factorioDir, constants.AdminlistName))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("admin list = %v, want %v", got, want)
	}
}

func TestWriteWhitelistIncludesOldestEligiblePlayer(t *testing.T) {
	factorioDir := configurePlayerListFiles(t)
	cfg.Local.Options.RegularsOnly = true
	glob.PlayerList = map[string]*glob.PlayerData{
		"oldest":    {Name: "oldest", Level: 2, LastSeen: 1},
		"newest":    {Name: "newest", Level: 2, LastSeen: 3},
		"member":    {Name: "member", Level: 1, LastSeen: 4},
		"admin-254": {Name: "admin-254", Level: 254, LastSeen: 6},
		"moderator": {Name: "moderator", Level: 255, LastSeen: 5},
	}

	if got := WriteWhitelist(); got != 4 {
		t.Fatalf("WriteWhitelist() = %d, want 4", got)
	}
	want := []string{"admin-254", "moderator", "newest", "oldest"}
	got := readPlayerNames(t, filepath.Join(factorioDir, constants.WhitelistName))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("whitelist = %v, want %v", got, want)
	}
}

func TestWriteWhitelistSinglePlayerIsValidJSON(t *testing.T) {
	factorioDir := configurePlayerListFiles(t)
	cfg.Local.Options.MembersOnly = true
	glob.PlayerList = map[string]*glob.PlayerData{
		"member": {Name: "member", Level: 1, LastSeen: 1},
	}

	if got := WriteWhitelist(); got != 1 {
		t.Fatalf("WriteWhitelist() = %d, want 1", got)
	}
	want := []string{"member"}
	got := readPlayerNames(t, filepath.Join(factorioDir, constants.WhitelistName))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("whitelist = %v, want %v", got, want)
	}
}

func TestPrivateWhitelistPreservesInvitesAndAddsStaff(t *testing.T) {
	root := configurePlayerListFiles(t)
	cfg.Local.Options.CustomWhitelist = true
	// Custom invitations take precedence over generated member/regular lists.
	cfg.Local.Options.RegularsOnly = true
	cfg.Local.Options.MembersOnly = true
	glob.PlayerList = map[string]*glob.PlayerData{
		"moderator": {Name: "moderator", Level: 255},
		"admin":     {Name: "admin", Level: 254},
		"regular":   {Name: "regular", Level: 2},
	}
	path := filepath.Join(root, constants.WhitelistName)
	if err := os.WriteFile(path, []byte(`["InvitedGuest","MODERATOR"]`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := WriteWhitelist(); got != 3 {
		t.Fatalf("count = %d, want 3", got)
	}
	want := []string{"InvitedGuest", "MODERATOR", "admin"}
	if got := readPlayerNames(t, path); !reflect.DeepEqual(got, want) {
		t.Fatalf("whitelist = %v, want %v", got, want)
	}
	if got := WriteWhitelist(); got != 3 {
		t.Fatalf("duplicate staff entries: count %d", got)
	}
}

func TestPrivateWhitelistMalformedFileIsNotOverwritten(t *testing.T) {
	root := configurePlayerListFiles(t)
	cfg.Local.Options.CustomWhitelist = true
	glob.PlayerList = map[string]*glob.PlayerData{"moderator": {Name: "moderator", Level: 255}}
	path := filepath.Join(root, constants.WhitelistName)
	original := []byte(`["guest",`)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	if got := WriteWhitelist(); got != -1 {
		t.Fatalf("count = %d, want -1", got)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatalf("file changed: %q, %v", got, err)
	}
}

func TestMemberWhitelistKeepsModeratorAndExcludesBanned(t *testing.T) {
	root := configurePlayerListFiles(t)
	cfg.Local.Options.CustomWhitelist = false
	cfg.Local.Options.MembersOnly = true
	cfg.Local.Options.RegularsOnly = false
	glob.PlayerList = map[string]*glob.PlayerData{
		"moderator": {Name: "moderator", Level: 255},
		"member":    {Name: "member", Level: 1},
		"banned":    {Name: "banned", Level: -1},
	}
	if got := WriteWhitelist(); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
	want := []string{"moderator", "member"}
	if got := readPlayerNames(t, filepath.Join(root, constants.WhitelistName)); !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}
}
