package fact

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
)

func TestBuildNewMapArgsNoGeneratorDefaultUsesPlainCreate(t *testing.T) {
	tests := []struct {
		name      string
		generator string
		preset    string
	}{
		{name: "blank generator and blank preset", generator: "", preset: ""},
		{name: "none generator and blank preset", generator: "none", preset: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			settings := resolveMapCreateSettings(tc.generator, tc.preset)
			got := buildNewMapArgs("save.zip", false, 0, settings)
			want := []string{"--create", "save.zip"}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected args: got %#v want %#v", got, want)
			}
		})
	}
}

func TestBuildNewMapArgsNoGeneratorDefaultPreset(t *testing.T) {
	settings := resolveMapCreateSettings(" NoNe ", "DEFAULT")
	got := buildNewMapArgs("save.zip", false, 0, settings)
	want := []string{"--create", "save.zip", "--preset", "default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %#v want %#v", got, want)
	}
}

func TestBuildNewMapArgsNoGeneratorNonDefaultPreset(t *testing.T) {
	settings := resolveMapCreateSettings("none", "RICH-RESOURCES")
	got := buildNewMapArgs("save.zip", false, 0, settings)
	want := []string{"--create", "save.zip", "--preset", "rich-resources"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %#v want %#v", got, want)
	}
}

func TestBuildNewMapArgsInvalidPresetUsesPlainCreate(t *testing.T) {
	settings := resolveMapCreateSettings("none", "not-a-preset")
	got := buildNewMapArgs("save.zip", false, 0, settings)
	want := []string{"--create", "save.zip"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %#v want %#v", got, want)
	}
}

func TestResolveMapCreateSettingsMissingCustomGeneratorFallsBackToNoGenerator(t *testing.T) {
	restoreMapGeneratorTestConfig(t)

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	cfg.Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	settings := resolveMapCreateSettings(constants.CustomMapGeneratorName, "default")
	if settings.usesGenerator() {
		t.Fatalf("expected missing custom generator to be ignored, got %#v", settings)
	}

	got := buildNewMapArgs("save.zip", false, 0, settings)
	want := []string{"--create", "save.zip", "--preset", "default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %#v want %#v", got, want)
	}
}

func TestResolveMapCreateSettingsCustomGeneratorUsesLocalFiles(t *testing.T) {
	restoreMapGeneratorTestConfig(t)

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	cfg.Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	genDir := filepath.Join(tmp, constants.DefaultMapGeneratorsDir)
	if err := os.MkdirAll(genDir, 0755); err != nil {
		t.Fatalf("failed creating generator directory: %v", err)
	}
	genPath := filepath.Join(genDir, constants.CustomMapGeneratorName+"-gen.json")
	setPath := filepath.Join(genDir, constants.CustomMapGeneratorName+"-set.json")
	if err := os.WriteFile(genPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing map-gen settings: %v", err)
	}
	if err := os.WriteFile(setPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing map settings: %v", err)
	}

	settings := resolveMapCreateSettings(constants.CustomMapGeneratorName, "rich-resources")
	if !settings.usesGenerator() {
		t.Fatalf("expected custom generator to be used, got %#v", settings)
	}

	got := buildNewMapArgs("save.zip", true, 123, settings)
	want := []string{
		"--create", "save.zip",
		"--map-gen-seed", "123",
		"--map-gen-settings", genPath,
		"--map-settings", setPath,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %#v want %#v", got, want)
	}
}

func TestResolveMapCreateSettingsMissingGeneratorUsesCachedCopy(t *testing.T) {
	restoreMapGeneratorTestConfig(t)

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	cfg.Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	genPath, setPath := cfg.GetCachedMapGeneratorFiles("spiral")
	if err := os.MkdirAll(filepath.Dir(genPath), 0755); err != nil {
		t.Fatalf("failed creating cached generator directory: %v", err)
	}
	if err := os.WriteFile(genPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing cached map-gen settings: %v", err)
	}
	if err := os.WriteFile(setPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed writing cached map settings: %v", err)
	}

	settings := resolveMapCreateSettings("spiral", "rich-resources")
	if !settings.usesGenerator() {
		t.Fatalf("expected cached generator to be used, got %#v", settings)
	}
	if !settings.usingCachedGenerator {
		t.Fatalf("expected usingCachedGenerator to be true, got %#v", settings)
	}
	if settings.mapGenSettingsPath != genPath {
		t.Fatalf("unexpected cached map-gen path: %q", settings.mapGenSettingsPath)
	}
	if settings.mapSettingsPath != setPath {
		t.Fatalf("unexpected cached map-settings path: %q", settings.mapSettingsPath)
	}
	if !strings.Contains(settings.fallbackNotice, "**MAP GENERATOR FALLBACK:**") {
		t.Fatalf("fallback notice is not clearly bold: %q", settings.fallbackNotice)
	}
}

func TestResolveMapCreateSettingsMissingGeneratorReportsPresetFallback(t *testing.T) {
	restoreMapGeneratorTestConfig(t)

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	cfg.Global.Paths.Folders.MapGenerators = constants.DefaultMapGeneratorsDir

	settings := resolveMapCreateSettings("spiral", "rich-resources")
	if settings.usesGenerator() {
		t.Fatalf("expected missing generator without cache to fall back, got %#v", settings)
	}
	if !strings.Contains(settings.fallbackNotice, "no local cached copy") {
		t.Fatalf("fallback notice does not mention missing cache: %q", settings.fallbackNotice)
	}
	if !strings.Contains(settings.fallbackNotice, `"rich-resources"`) {
		t.Fatalf("fallback notice does not mention preset: %q", settings.fallbackNotice)
	}
}

func TestAnnounceMapGeneratorFallbackQueuesBoldDiscordNotice(t *testing.T) {
	drainMapResetCMS()
	t.Cleanup(drainMapResetCMS)

	oldChannel := cfg.Local.Channel.ChatChannel
	t.Cleanup(func() { cfg.Local.Channel.ChatChannel = oldChannel })
	cfg.Local.Channel.ChatChannel = "chan-1"

	notice := "**MAP GENERATOR FALLBACK:** Configured generator missing."
	announceMapGeneratorFallback(notice)

	select {
	case queued := <-disc.CMSChan:
		if queued.Channel != "chan-1" {
			t.Fatalf("unexpected queued channel: %q", queued.Channel)
		}
		if queued.Text != notice {
			t.Fatalf("unexpected queued notice: %q", queued.Text)
		}
	default:
		t.Fatal("expected fallback notice to be queued")
	}
}

func drainMapResetCMS() {
	for {
		select {
		case <-disc.CMSChan:
		default:
			return
		}
	}
}

func restoreMapGeneratorTestConfig(t *testing.T) {
	t.Helper()

	oldGlobal := cfg.Global
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed reading working directory: %v", err)
	}

	t.Cleanup(func() {
		cfg.Global = oldGlobal
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed restoring working directory: %v", err)
		}
	})
}
