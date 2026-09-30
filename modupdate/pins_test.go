package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modedit"
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pinnedModZip(t *testing.T, name, version string, deps []string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, err := z.Create(name + "_" + version + "/info.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(f).Encode(modZipInfo{Name: name, Version: version, FactorioVersion: "2.0", Dependencies: deps}); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestPinnedDependencyPreventsIncompatibleUpdate(t *testing.T) {
	oldGlobal, oldLocal, oldProxy, oldVersion := cfg.Global, cfg.Local, glob.ProxyURL, fact.FactorioVersion
	t.Cleanup(func() {
		cfg.Global, cfg.Local, glob.ProxyURL, fact.FactorioVersion = oldGlobal, oldLocal, oldProxy, oldVersion
	})
	root := t.TempDir()
	t.Chdir(root)
	cfg.Global.Paths.Folders.ServersRoot = root + "/"
	cfg.Global.Paths.ChatWirePrefix = "cw-"
	cfg.Global.Paths.Folders.FactorioDir = "factorio"
	cfg.Local.Callsign = "a"
	cfg.Local.Channel.ChatChannel = ""
	fact.FactorioVersion = "2.0.76"
	mods := cfg.GetModsFolder()
	if err := os.MkdirAll(mods, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"parent", "dep"} {
		if err := os.WriteFile(filepath.Join(mods, name+"_1.0.0.zip"), pinnedModZip(t, name, "1.0.0", nil), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if !WriteModsList(ModListData{Mods: []ModData{{Name: "parent", Enabled: true}, {Name: "dep", Enabled: true}}}) {
		t.Fatal("write mod list")
	}
	if err := modedit.SetVersion("dep", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	archives := map[string][]byte{}
	portal := map[string]modPortalFullData{}
	for _, name := range []string{"parent", "dep"} {
		info := modPortalFullData{Name: name, Title: name}
		for _, version := range []string{"1.0.0", "2.0.0"} {
			var deps []string
			if name == "parent" && version == "2.0.0" {
				deps = []string{"dep >= 2.0.0"}
			}
			data := pinnedModZip(t, name, version, deps)
			hash := sha1.Sum(data)
			filename := name + "_" + version + ".zip"
			archives["/download/"+filename] = data
			info.Releases = append(info.Releases, modRelease{Version: version, FileName: filename, DownloadURL: "/download/" + filename, Sha1: hex.EncodeToString(hash[:]), InfoJSON: modInfoJSON{FactorioVersion: "2.0", Dependencies: deps}})
		}
		portal[name] = info
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := strings.TrimPrefix(r.URL.Path, "/")
		for name, info := range portal {
			if target == "https://mods.factorio.com/api/mods/"+name+"/full" {
				data, _ := json.Marshal(info)
				w.Write(data)
				return
			}
		}
		for path, data := range archives {
			if strings.HasPrefix(target, "https://mods.factorio.com"+path+"?") {
				w.Write(data)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	proxy := server.URL
	glob.ProxyURL = &proxy
	updated, err := CheckModUpdates(false, false, true, true)
	if err != nil && err.Error() != "no mod updates available" {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("incompatible parent update was selected")
	}
	files, err := GetModFiles()
	if err != nil {
		t.Fatal(err)
	}
	versions := map[string]string{}
	for _, m := range files {
		versions[m.Name] = m.Version
	}
	if versions["parent"] != "1.0.0" || versions["dep"] != "1.0.0" {
		t.Fatalf("update reported success=%t but installed parent 2.0.0 requires dep >=2.0.0; pinned dep is still %s", updated, versions["dep"])
	}
}

func TestResolveDepsHonorsPinnedVersions(t *testing.T) {
	old := fact.FactorioVersion
	fact.FactorioVersion = "2.0.76"
	t.Cleanup(func() { fact.FactorioVersion = old })
	for _, tc := range []struct {
		name, installed, pin, factorio, want string
		fail                                 bool
	}{
		{name: "downgrade", installed: "2.0.0", pin: "1.0.0", factorio: "2.0", want: "1.0.0"},
		{name: "upgrade", installed: "1.0.0", pin: "2.0.0", factorio: "2.0", want: "2.0.0"},
		{name: "already pinned", installed: "1.0.0", pin: "1.0.0", factorio: "2.0"},
		{name: "automatic", installed: "1.0.0", pin: "AUTO", factorio: "2.0", want: "2.0.0"},
		{name: "missing release", installed: "1.0.0", pin: "3.0.0", factorio: "2.0", fail: true},
		{name: "wrong Factorio version", installed: "2.0.0", pin: "1.0.0", factorio: "1.1", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := modPortalFullData{Name: "mod", installed: modZipInfo{Version: tc.installed}, Releases: []modRelease{
				{Version: "1.0.0", InfoJSON: modInfoJSON{FactorioVersion: tc.factorio}},
				{Version: "2.0.0", InfoJSON: modInfoJSON{FactorioVersion: "2.0"}},
			}}
			prefs := modedit.VersionPrefs{Mods: []modedit.ModVersion{{Name: "mod", Version: tc.pin}}}
			downloads, err := resolveDepsWithPrefs([]modPortalFullData{info}, false, 0, nil, prefs, nil)
			if (err != nil) != tc.fail {
				t.Fatalf("downloads=%+v error=%v", downloads, err)
			}
			if tc.fail {
				return
			}
			if tc.want == "" {
				if len(downloads) != 0 {
					t.Fatalf("unexpected downloads=%+v", downloads)
				}
				return
			}
			if len(downloads) != 1 || downloads[0].Version != tc.want {
				t.Fatalf("downloads=%+v want version=%s", downloads, tc.want)
			}
		})
	}
}

func TestPinnedInstalledReleaseResolvesDependencies(t *testing.T) {
	old := fact.FactorioVersion
	fact.FactorioVersion = "2.0.76"
	t.Cleanup(func() { fact.FactorioVersion = old })
	portal := []modPortalFullData{
		{Name: "parent", installed: modZipInfo{Version: "1.0.0"}, Releases: []modRelease{{Version: "1.0.0", InfoJSON: modInfoJSON{FactorioVersion: "2.0", Dependencies: []string{"dep >= 2.0.0"}}}}},
		{Name: "dep", installed: modZipInfo{Version: "1.0.0"}, Releases: []modRelease{{Version: "2.0.0", InfoJSON: modInfoJSON{FactorioVersion: "2.0"}}}},
	}
	prefs := modedit.VersionPrefs{Mods: []modedit.ModVersion{{Name: "parent", Version: "1.0.0"}}}
	downloads, err := resolveDepsWithPrefs(portal, false, 0, nil, prefs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(downloads) != 1 || downloads[0].Name != "dep" || downloads[0].Version != "2.0.0" {
		t.Fatalf("pinned parent dependencies=%+v", downloads)
	}
}

func TestValidateFinalModDownloadPlan(t *testing.T) {
	old := fact.FactorioVersion
	fact.FactorioVersion = "2.0.76"
	t.Cleanup(func() { fact.FactorioVersion = old })
	installed := func(name, version string, deps ...string) modZipInfo {
		return modZipInfo{Name: name, Version: version, Dependencies: deps, Enabled: true}
	}
	planned := func(name, version string, deps ...string) downloadData {
		return downloadData{Name: name, Version: version, Data: modRelease{Version: version, InfoJSON: modInfoJSON{Dependencies: deps}}}
	}
	for _, tc := range []struct {
		name      string
		installed []modZipInfo
		downloads []downloadData
		fail      bool
	}{
		{name: "pinned dependency too old", installed: []modZipInfo{installed("dep", "1.0.0")}, downloads: []downloadData{planned("parent", "2.0.0", "dep >= 2.0.0")}, fail: true},
		{name: "compatible dependency", installed: []modZipInfo{installed("dep", "2.0.0")}, downloads: []downloadData{planned("parent", "2.0.0", "dep >= 2.0.0")}},
		{name: "other enabled mod rejects merged upgrade", installed: []modZipInfo{installed("parent", "1.0.0", "dep < 2.0.0"), installed("dep", "1.0.0")}, downloads: []downloadData{planned("dep", "2.0.0")}, fail: true},
		{name: "replacement removes old incompatibility", installed: []modZipInfo{installed("parent", "1.0.0", "! dep"), installed("dep", "1.0.0")}, downloads: []downloadData{planned("parent", "2.0.0")}},
		{name: "missing dependency", downloads: []downloadData{planned("parent", "2.0.0", "dep")}, fail: true},
		{name: "missing optional dependency", downloads: []downloadData{planned("parent", "2.0.0", "? dep >= 2.0.0")}},
		{name: "present optional dependency too old", installed: []modZipInfo{installed("dep", "1.0.0")}, downloads: []downloadData{planned("parent", "2.0.0", "? dep >= 2.0.0")}, fail: true},
		{name: "incompatible dependency", installed: []modZipInfo{installed("dep", "1.0.0")}, downloads: []downloadData{planned("parent", "2.0.0", "! dep")}, fail: true},
		{name: "Factorio dependency", downloads: []downloadData{planned("parent", "2.0.0", "base >= 2.0.0")}},
		{name: "wrong Factorio dependency", downloads: []downloadData{planned("parent", "2.0.0", "base >= 3.0.0")}, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDownloadPlan(tc.installed, tc.downloads); (err != nil) != tc.fail {
				t.Fatalf("plan validation error=%v want failure=%t", err, tc.fail)
			}
		})
	}
}
