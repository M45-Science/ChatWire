package modupdate

import (
	"testing"

	"ChatWire/fact"
)

func TestDependencySatisfiedUnversionedDepPresent(t *testing.T) {
	good, err := dependencySatisfied(depRequires{name: "dep"}, "1.0.0", nil)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if !good {
		t.Fatal("expected unversioned installed dependency to satisfy requirement")
	}
}

func TestDependencySatisfiedUnversionedDepMissing(t *testing.T) {
	good, err := dependencySatisfied(depRequires{name: "dep"}, "", nil)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if good {
		t.Fatal("expected missing unversioned dependency to fail")
	}
}

func TestDependencySatisfiedVersionedDepSatisfiedByInstalledVersion(t *testing.T) {
	good, err := dependencySatisfied(depRequires{name: "dep", equality: EO_GREATEREQ, version: "1.2.0"}, "1.3.0", nil)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if !good {
		t.Fatal("expected installed version to satisfy dependency")
	}
}

func TestDependencySatisfiedVersionedDepSatisfiedByPlannedUpdate(t *testing.T) {
	planned := []downloadData{{Name: "dep", Version: "1.4.0"}}
	good, err := dependencySatisfied(depRequires{name: "dep", equality: EO_GREATEREQ, version: "1.2.0"}, "1.0.0", planned)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if !good {
		t.Fatal("expected planned download version to satisfy dependency")
	}
}

func TestDependencySatisfiedVersionedDepUnsatisfiedAfterRecursion(t *testing.T) {
	planned := []downloadData{{Name: "dep", Version: "1.1.0"}}
	good, err := dependencySatisfied(depRequires{name: "dep", equality: EO_GREATEREQ, version: "1.2.0"}, "1.0.0", planned)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if good {
		t.Fatal("expected dependency to remain unsatisfied after recursion")
	}
}

func TestResolveDepsDropsSupersededReleaseDependencies(t *testing.T) {
	oldFactorioVersion := fact.FactorioVersion
	oldDownloadModInfo := downloadModInfo
	t.Cleanup(func() {
		fact.FactorioVersion = oldFactorioVersion
		downloadModInfo = oldDownloadModInfo
	})

	fact.FactorioVersion = "2.0.76"
	downloadModInfo = func(name string) (modPortalFullData, error) {
		return modPortalFullData{
			Name:  name,
			Title: name,
			Releases: []modRelease{
				{
					Version:  "1.0.0",
					FileName: name + "_1.0.0.zip",
					InfoJSON: modInfoJSON{
						FactorioVersion: "2.0",
					},
				},
			},
		}, nil
	}

	downloads, err := resolveDeps([]modPortalFullData{
		{
			Name:  "parent",
			Title: "Parent",
			Releases: []modRelease{
				{
					Version:  "1.0.0",
					FileName: "parent_1.0.0.zip",
					InfoJSON: modInfoJSON{
						FactorioVersion: "2.0",
						Dependencies:    []string{"dep >= 1.0.0"},
					},
				},
				{
					Version:  "2.0.0",
					FileName: "parent_2.0.0.zip",
					InfoJSON: modInfoJSON{
						FactorioVersion: "2.0",
					},
				},
			},
		},
	}, false, 0, nil, nil)
	if err != nil {
		t.Fatalf("resolveDeps returned error: %v", err)
	}
	if len(downloads) != 1 {
		t.Fatalf("expected only final parent download, got %+v", downloads)
	}
	if downloads[0].Name != "parent" || downloads[0].Version != "2.0.0" {
		t.Fatalf("unexpected selected download: %+v", downloads[0])
	}
}

func TestResolveDepsTagsFinalDependencyRequester(t *testing.T) {
	oldFactorioVersion := fact.FactorioVersion
	oldDownloadModInfo := downloadModInfo
	t.Cleanup(func() {
		fact.FactorioVersion = oldFactorioVersion
		downloadModInfo = oldDownloadModInfo
	})

	fact.FactorioVersion = "2.0.76"
	downloadModInfo = func(name string) (modPortalFullData, error) {
		return modPortalFullData{
			Name:  name,
			Title: name,
			Releases: []modRelease{
				{
					Version:  "1.0.0",
					FileName: name + "_1.0.0.zip",
					InfoJSON: modInfoJSON{
						FactorioVersion: "2.0",
					},
				},
			},
		}, nil
	}

	downloads, err := resolveDeps([]modPortalFullData{
		{
			Name:  "parent",
			Title: "Parent",
			Releases: []modRelease{
				{
					Version:  "2.0.0",
					FileName: "parent_2.0.0.zip",
					InfoJSON: modInfoJSON{
						FactorioVersion: "2.0",
						Dependencies:    []string{"dep >= 1.0.0"},
					},
				},
			},
		},
	}, false, 0, nil, nil)
	if err != nil {
		t.Fatalf("resolveDeps returned error: %v", err)
	}

	for _, dl := range downloads {
		if dl.Name == "dep" {
			if !dl.wasDep {
				t.Fatalf("expected dep download to be marked as dependency: %+v", dl)
			}
			if dl.RequiredByName != "parent" || dl.RequiredByVersion != "2.0.0" {
				t.Fatalf("expected dependency requester parent-2.0.0, got %+v", dl)
			}
			return
		}
	}
	t.Fatalf("expected dependency download in %+v", downloads)
}

func TestResolveDepsSkipsWrongFactorioVersionRelease(t *testing.T) {
	oldFactorioVersion := fact.FactorioVersion
	t.Cleanup(func() {
		fact.FactorioVersion = oldFactorioVersion
	})

	fact.FactorioVersion = "2.0.76"
	downloads, err := resolveDeps([]modPortalFullData{
		{
			Name:  "stdlib",
			Title: "Factorio Standard Library",
			Releases: []modRelease{
				{
					Version:  "1.4.8",
					FileName: "stdlib_1.4.8.zip",
					InfoJSON: modInfoJSON{
						FactorioVersion: "1.1",
						Dependencies:    []string{"base >= 1.1.0"},
					},
				},
			},
		},
	}, true, 0, nil, nil)
	if err != nil {
		t.Fatalf("resolveDeps returned error: %v", err)
	}
	if len(downloads) != 0 {
		t.Fatalf("expected no downloads for incompatible factorio_version, got %+v", downloads)
	}
}
