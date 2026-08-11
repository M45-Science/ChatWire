package modupdate

import "testing"

func TestIsBaseMod(t *testing.T) {
	bundledMods := []string{"base", "elevated-rails", "quality", "space-age", "recycler"}
	for _, name := range bundledMods {
		if !IsBaseMod(name) {
			t.Errorf("expected %q to be treated as a bundled mod", name)
		}
	}

	if IsBaseMod("some-portal-mod") {
		t.Error("expected a portal mod not to be treated as bundled")
	}
}

func TestGetCompletedDownloadCount(t *testing.T) {
	downloads := []downloadData{
		{Name: "a", Complete: true},
		{Name: "b", Complete: false},
		{Name: "c", Complete: true},
	}

	if got := getCompletedDownloadCount(downloads); got != 2 {
		t.Fatalf("expected 2 completed downloads, got %d", got)
	}
}

func TestResetModInfoCacheClearsEntries(t *testing.T) {
	ModInfoLock.Lock()
	modInfoCache = map[string]modPortalFullData{
		"dep": {Name: "dep"},
	}
	ModInfoLock.Unlock()

	resetModInfoCache()

	ModInfoLock.Lock()
	defer ModInfoLock.Unlock()
	if len(modInfoCache) != 0 {
		t.Fatalf("expected empty mod info cache after reset, got %d entries", len(modInfoCache))
	}
}
