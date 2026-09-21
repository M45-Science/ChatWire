package webcontrol

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestSocketOwnershipAndStaleRecovery(t *testing.T) {
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "control.sock")
	first, e := Listen(path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Listen(path); e == nil {
		t.Fatal("stole live socket")
	}
	first.Close()
	second, e := Listen(path)
	if e != nil {
		t.Fatal(e)
	}
	second.Close()
	l, e := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if e != nil {
		t.Fatal(e)
	}
	l.SetUnlinkOnClose(false)
	l.Close()
	third, e := Listen(path)
	if e != nil {
		t.Fatalf("stale socket not reclaimed: %v", e)
	}
	third.Close()
	if e = os.WriteFile(path, []byte("not a socket"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Listen(path); e == nil {
		t.Fatal("deleted an unrelated file")
	}
}
func TestCredentialPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key")
	if e := os.WriteFile(path, []byte(Token()), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := Credential(path); e == nil {
		t.Fatal("accepted world-readable credential")
	}
	os.Chmod(path, 0600)
	if _, e := Credential(path); e != nil {
		t.Fatal(e)
	}
}
