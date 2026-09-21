package controlruntime

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func zipFixture(t *testing.T, entries []string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "test.zip")
	f, e := os.Create(name)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	for _, n := range entries {
		w, e := z.Create(n)
		if e != nil {
			t.Fatal(e)
		}
		w.Write([]byte("test"))
	}
	z.Close()
	f.Close()
	return name
}
func TestUploadSaveValidation(t *testing.T) {
	for _, tc := range []struct {
		entries []string
		valid   bool
	}{{[]string{"world/level.dat"}, true}, {[]string{"../evil", "world/level.dat"}, false}, {[]string{"/etc/evil", "world/level.dat"}, false}, {[]string{"world/other.txt"}, false}} {
		if e := validateUpload(zipFixture(t, tc.entries), "save"); (e == nil) != tc.valid {
			t.Fatalf("%v: %v", tc.entries, e)
		}
	}
}
func TestSaveIDsRejectTraversal(t *testing.T) {
	for _, name := range []string{"../save.zip", "/save.zip", "..\\save.zip", "not-a-save"} {
		if _, e := saveName(saveID(name)); e == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if name, e := saveName(saveID("normal save.zip")); e != nil || name != "normal save.zip" {
		t.Fatal(name, e)
	}
}
