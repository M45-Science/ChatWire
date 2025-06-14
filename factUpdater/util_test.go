package factUpdater

import (
	"archive/tar"
	"bytes"
	"testing"
)

// helper to build a simple tar archive from a list of file names
func buildTar(files []string) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, name := range files {
		hdr := &tar.Header{Name: name, Mode: 0600, Size: int64(len(name))}
		tw.WriteHeader(hdr)
		tw.Write([]byte(name))
	}
	tw.Close()
	return buf.Bytes()
}

func TestCheckInstallTar(t *testing.T) {
	files := []string{
		"factorio/data/eula.txt",
		"factorio/data/licenses.txt",
		"factorio/data/credits.txt",
		"factorio/data/changelog.txt",
		"factorio/data/core/backers.json",
		"factorio/data/core/data.lua",
		"factorio/data/core/info.json",
	}
	if err := checkInstallTar(buildTar(files)); err != nil {
		t.Fatalf("valid archive: %v", err)
	}
	bad := files[:len(files)-1]
	if err := checkInstallTar(buildTar(bad)); err == nil {
		t.Fatalf("expected error when file missing")
	}
}

func TestParseVersionString(t *testing.T) {
	v, err := parseVersionString("1.2.3")
	if err != nil || v.A != 1 || v.B != 2 || v.C != 3 {
		t.Fatalf("unexpected: %v %v", v, err)
	}
	if _, err := parseVersionString("1.2"); err == nil {
		t.Fatalf("expected error on short version")
	}
}

func TestVersionCompare(t *testing.T) {
	a := versionInts{A: 1, B: 2, C: 3}
	b := versionInts{A: 1, B: 2, C: 2}
	if !isVersionNewerThan(a, b) {
		t.Fatalf("expected newer")
	}
	if isVersionNewerThan(b, a) {
		t.Fatalf("expected older")
	}
	if !isVersionEqual(a, versionInts{1, 2, 3}) {
		t.Fatalf("equality failed")
	}
}

func TestParseFactorioVersion(t *testing.T) {
	input := "Version: 1.1.90 (build 12345, linux64, headless)"
	v, bld, dist, err := parseFactorioVersion(input)
	if err != nil || v.A != 1 || v.C != 90 || bld != "headless" || dist != "linux64" {
		t.Fatalf("unexpected output: %v %v %v %v", v, bld, dist, err)
	}
}
