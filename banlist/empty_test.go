package banlist

import (
	"ChatWire/cfg"
	"os"
	"path/filepath"
	"testing"
)

func TestEmptyBanFileRevokesFinalBan(t *testing.T) {
	oldPath, oldList := cfg.Global.Paths.DataFiles.Bans, BanList
	t.Cleanup(func() { cfg.Global.Paths.DataFiles.Bans, BanList = oldPath, oldList })
	path := filepath.Join(t.TempDir(), "bans.json")
	cfg.Global.Paths.DataFiles.Bans = path
	BanList = []banDataType{{UserName: "alice", Reason: "test"}}
	if err := os.WriteFile(path, []byte(`[]`), 0600); err != nil {
		t.Fatal(err)
	}
	ReadBanFile(true)
	for _, b := range BanList {
		if b.UserName == "alice" && !b.Revoked {
			t.Fatal("valid empty ban list kept the final revoked ban active")
		}
	}
}

func TestInvalidBanFilePreservesActiveBans(t *testing.T) {
	oldPath, oldList := cfg.Global.Paths.DataFiles.Bans, BanList
	t.Cleanup(func() { cfg.Global.Paths.DataFiles.Bans, BanList = oldPath, oldList })
	path := filepath.Join(t.TempDir(), "bans.json")
	cfg.Global.Paths.DataFiles.Bans = path
	for _, data := range []string{"null", "", `[broken`} {
		BanList = []banDataType{{UserName: "alice", Reason: "test"}}
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		ReadBanFile(true)
		if len(BanList) != 1 || BanList[0].Revoked {
			t.Fatalf("invalid data %q revoked active ban", data)
		}
	}
}
