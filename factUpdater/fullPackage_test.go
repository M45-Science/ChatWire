package factUpdater

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
	"github.com/ulikunitz/xz"
)

type testRoundTrip func(*http.Request) (*http.Response, error)

func (f testRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRegressionUpdateDoesNotInstallBeforeLifecycleStops(t *testing.T) {
	oldLocal, oldGlobal, oldProxy := cfg.Local, cfg.Global, glob.ProxyURL
	t.Cleanup(func() { cfg.Local, cfg.Global, glob.ProxyURL = oldLocal, oldGlobal, oldProxy })
	root := t.TempDir()
	binary := filepath.Join(root, "factorio/bin/x64/factorio")
	if err := os.MkdirAll(filepath.Dir(binary), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("original binary"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg.Global.Paths.Binaries.FactBinary = binary
	glob.ProxyURL = new(string)

	archive := testInstallArchive(t, &tar.Header{Name: "factorio/bin/x64/factorio", Mode: 0755, Size: 1})
	var xzbuf bytes.Buffer
	xw, err := xz.NewWriter(&xzbuf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := xw.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	download := xzbuf.Bytes()
	hash := sha256.Sum256(download)
	checksums := []byte(fmt.Sprintf("%x  factorio-headless_linux_2.0.76.tar.xz", hash))
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	http.DefaultTransport = testRoundTrip(func(r *http.Request) (*http.Response, error) {
		data := download
		if strings.Contains(r.URL.Path, "sha256sums") {
			data = checksums
		} else if r.URL.Path != "/get-download/2.0.76/headless/linux64" {
			t.Errorf("download URL does not identify the selected release: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: r, ContentLength: int64(len(data)), Body: io.NopCloser(bytes.NewReader(data))}, nil
	})

	// Virtual time exercises the production stop wait without a live process
	// or network requests.
	synctest.Test(t, func(t *testing.T) {
		fact.SetAutolaunch(false, false)
		fact.SetUpdateInProgress(false)
		fact.SetModOperationInProgress(false)
		entered, release := make(chan struct{}), make(chan struct{})
		fact.StartLifecycleManager(fact.LifecycleHooks{LaunchFactorio: func(uint64, string) error {
			close(entered)
			<-release
			return fmt.Errorf("audit launch canceled")
		}})
		defer fact.StopLifecycleManager()
		defer close(release)
		if err := fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionStart}); err != nil {
			t.Fatal(err)
		}
		<-entered
		info := &InfoData{Build: "headless", Distro: "linux64", VersInt: versionInts{A: 2, B: 0, C: 76}}
		if err := fullPackage(info, false); err == nil || !strings.Contains(err.Error(), "waiting for Factorio to stop") {
			t.Fatalf("expected installation to abort on stop timeout, got %v", err)
		}
		state := fact.GetLifecycleState()
		got, err := os.ReadFile(binary)
		if err != nil {
			t.Fatal(err)
		}
		if state.Phase != fact.LifecycleStopped && string(got) != "original binary" {
			t.Errorf("installer replaced Factorio while lifecycle phase=%s; stop wait timeout was ignored", state.Phase)
		}
	})
}

func TestQuickLatestReadsSelectedRelease(t *testing.T) {
	for _, tc := range []struct {
		name, stable, experimental, want string
		useExperimental                  bool
	}{
		{"stable with missing experimental", "2.0.76", "", "2.0.76", false},
		{"stable with malformed experimental", "2.0.76", "broken", "2.0.76", false},
		{"experimental with malformed stable", "broken", "2.1.24", "2.1.24", true},
		{"missing selected release", "2.0.76", "", "", true},
		{"malformed selected release", "broken", "2.1.24", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldProxy, oldTransport := glob.ProxyURL, http.DefaultTransport
			t.Cleanup(func() { glob.ProxyURL, http.DefaultTransport = oldProxy, oldTransport })
			glob.ProxyURL = new(string)
			body := fmt.Sprintf(`{"stable":{"headless":%q},"experimental":{"headless":%q}}`, tc.stable, tc.experimental)
			http.DefaultTransport = testRoundTrip(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Request: r, ContentLength: int64(len(body)), Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			version, err := quickLatest(&InfoData{Xreleases: tc.useExperimental})
			if tc.want == "" {
				if err == nil || version != nil {
					t.Fatalf("expected an error, got %v, %v", version, err)
				}
			} else if err != nil || version == nil || version.IntToString() != tc.want {
				t.Fatalf("wanted %s, got %v, %v", tc.want, version, err)
			}
		})
	}
}

func testInstallArchive(t *testing.T, binary *tar.Header) []byte {
	t.Helper()
	var tarbuf bytes.Buffer
	tw := tar.NewWriter(&tarbuf)
	for _, dir := range []string{"factorio/", "factorio/bin/", "factorio/bin/x64/", "factorio/data/", "factorio/data/core/"} {
		if err := tw.WriteHeader(&tar.Header{Name: dir, Mode: 0755, Typeflag: tar.TypeDir}); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"factorio/data/eula.txt", "factorio/data/licenses.txt", "factorio/data/credits.txt", "factorio/data/changelog.txt", "factorio/data/core/backers.json", "factorio/data/core/data.lua", "factorio/data/core/info.json"} {
		data := []byte("replacement binary or data")
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if binary != nil {
		if err := tw.WriteHeader(binary); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(make([]byte, binary.Size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return tarbuf.Bytes()
}

func TestCheckInstallTarRequiresRunnableBinary(t *testing.T) {
	const name = "factorio/bin/x64/factorio"
	for _, tc := range []struct {
		name   string
		header *tar.Header
	}{
		{"missing", nil},
		{"empty", &tar.Header{Name: name, Mode: 0755}},
		{"non-executable", &tar.Header{Name: name, Mode: 0644, Size: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkInstallTar(testInstallArchive(t, tc.header)); err == nil {
				t.Fatal("install package without a usable binary passed verification")
			}
		})
	}
}
