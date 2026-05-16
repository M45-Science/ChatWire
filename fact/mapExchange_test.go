package fact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ChatWire/cfg"
	"ChatWire/constants"
)

const sampleMapExchangeString = `>>>eNpjYmBg8AFiBh6W5PzEHAaGBnsY5krOLyhILdLNL0pFFuZMLipNSdXNz0RVnJqXmlupm5RYDFUMAg32HJlF+XnoJvDkJZaWZRbHJ+dkpqUhS7AW5SdnFyOLiBWXJBaVZOalxycWpSbG5+ZnFpeUoprGWlySn4cqUlKUmopiDHdpUWJeZmkuuktYyxNLUosgPAeIKKOn9iTzhhY5BhD+X8+g8P8/CANZD4A+AmEGxgaIDkagIAywQj3DoOAIxE5wSxgZGKtF1rk/rJpizwhRqecAZXyAihxIgol4whh+DjilVGAMEyRzjMHgMxIDYmkJ0AqoKg4HBAMi2QKSZGTsfbt1wfdjF+wY/6z8eMk3KcGe0dBV5N0Ho3V2QEl2kHeZ4MSsmSCwE+YVBpiZD+yhUjftGc+eAYE39oysIB0iIMLBAkgc8GZmYBTgA7IW9AAJBRkGmNPsYMaIODCmgcE3mE8ewxiX7dH9AQwIG5DhciDiBIgAWwh3GSOE6dDvwOggD5OVRCgB6jdiQHZDCsKHJ2HWHkayH80hmBGB7A80ERUHLNHABbIwBU68YIa7BhieF9hhPIf5DozMIAZI1RegGIQHkoEZBaEFHMDBzQxPlB/sUVMaiAEyJP29ZCQA2Pi9LQ==<<<`

func TestParseMapExchangeString(t *testing.T) {
	data, err := ParseMapExchangeString(sampleMapExchangeString)
	if err != nil {
		t.Fatalf("ParseMapExchangeString returned error: %v", err)
	}

	if data.Version[0] == 0 {
		t.Fatalf("expected non-zero Factorio version, got %v", data.Version)
	}
	if !data.ChecksumOK {
		t.Fatal("expected checksum to validate")
	}
	if len(data.MapGenSettings) == 0 {
		t.Fatal("expected map gen settings")
	}
	if len(data.MapSettings) == 0 {
		t.Fatal("expected map settings")
	}
	if _, ok := data.MapGenSettings["autoplace_controls"]; !ok {
		t.Fatal("expected autoplace_controls in map gen settings")
	}
	if _, ok := data.MapSettings["pollution"]; !ok {
		t.Fatal("expected pollution in map settings")
	}
}

func TestParseMapExchangeJSON(t *testing.T) {
	data, err := ParseMapExchangeString(`{"map_settings":{"pollution":{"enabled":true}},"map_gen_settings":{"seed":123}}`)
	if err != nil {
		t.Fatalf("ParseMapExchangeString returned error: %v", err)
	}
	if got := data.MapGenSettings["seed"].(json.Number).String(); got != "123" {
		t.Fatalf("unexpected seed: %v", got)
	}
}

func TestWriteCustomMapExchangeFiles(t *testing.T) {
	oldRoot := cfg.Global.Paths.Folders.ServersRoot
	oldMapGenerators := cfg.Global.Paths.Folders.MapGenerators
	t.Cleanup(func() {
		cfg.Global.Paths.Folders.ServersRoot = oldRoot
		cfg.Global.Paths.Folders.MapGenerators = oldMapGenerators
	})

	tmp := t.TempDir()
	cfg.Global.Paths.Folders.ServersRoot = tmp + string(os.PathSeparator)
	cfg.Global.Paths.Folders.MapGenerators = "map-gen-json"

	genPath, setPath, err := WriteCustomMapExchangeFiles(sampleMapExchangeString)
	if err != nil {
		t.Fatalf("WriteCustomMapExchangeFiles returned error: %v", err)
	}

	wantGen := filepath.Join(tmp, "map-gen-json", constants.CustomMapGeneratorName+"-gen.json")
	wantSet := filepath.Join(tmp, "map-gen-json", constants.CustomMapGeneratorName+"-set.json")
	if genPath != wantGen {
		t.Fatalf("unexpected gen path: got %q want %q", genPath, wantGen)
	}
	if setPath != wantSet {
		t.Fatalf("unexpected set path: got %q want %q", setPath, wantSet)
	}

	for _, path := range []string{genPath, setPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed reading %s: %v", path, err)
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("invalid JSON in %s: %v", path, err)
		}
		if len(parsed) == 0 {
			t.Fatalf("expected JSON payload in %s", path)
		}
	}
}
