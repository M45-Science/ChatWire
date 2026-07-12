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

func TestParseFactorio21MapExchangeString(t *testing.T) {
	input := `>>>eNp1VE2LEzEYTqx1u19apAjisvawN1nBVcGDdKIgIqJ/YUynaQ1OJzUfldWDPXhUvHjRi3sSXGE9eS+IordFz8KKFz0oFUQvQk06k2nSroFknnk/nud93wmzB0AwCwoALJXxTUVjFkZcNUjIaAxAL7B7JsJxRCVxbXsjhp0ggOYi1ukQvso4cc2zI8ZVj1EHk4S011frWHiki81YMU4TEnZJIn2PiluM4zCKabOZ0QDzPGA9VMQ4aQgrYZ7zrZjUd8kpp/ZREeFkEQups6PZZMaGAodNSJZ4CZn9FpaEu/YS5SyZnMdiTOV1qtph3fTp6SZYdamYrrbIWXRDuKFFEXHccS2HhMRc0qQVYk5w2GZUSMWJn+QXDlBFqLipOI1CHNFG2CLrwu+gKDkhnvKCVElLSJKEzGefVxwnuq+pfrsqjnCidF/ehekFB3NPlxlARfvF5uZYe2qeAB478ull794yMHt4F1SHQ7M12tFSZgPYA6Cvo6E22lXMJgqq5/Q+P6aD8E5l6+Ln248CmEYeRxkYZJZ+3VouWXAV/de1YsEphwfCB99fbfx5u12Df5///HClfi2AJy5UfgzWtmraWTRFF8yxL0dp7o6tqoQmLRo8eWzWtyAlqIzTygiiM/qtf7kAYHm/Rhv39VFdyiNqlq+CYHO0fgfw5Gh9seDjlLhu7qxRWTbHO3MUx6IA6VZS8FDrH7Xew+MQnb8G3Boa5i1t472Vfe3oTxQyPVy3jwlLHlxywJwRbOTH14I76u0Z+4aeotFnACbqF8w+Ckj/KSlVPmkz90J+0VaQf3sMMCRvnp0e/AMndh/V<<<`

	data, err := ParseMapExchangeString(input)
	if err != nil {
		t.Fatalf("ParseMapExchangeString returned error: %v", err)
	}
	if data.Version != [4]uint16{2, 1, 9, 3} {
		t.Fatalf("version = %v, want 2.1.9-3", data.Version)
	}
	if _, ok := data.MapSettings["steering"]; ok {
		t.Fatalf("Factorio 2.1 map-settings unexpectedly included removed steering block")
	}
	expansion := data.MapSettings["enemy_expansion"].(map[string]interface{})
	if got := expansion["min_expansion_distance"]; got != uint32(3) {
		t.Fatalf("min_expansion_distance = %#v, want 3", got)
	}
	if got := expansion["evolution_group_size_factor"]; got != float64(4) {
		t.Fatalf("evolution_group_size_factor = %#v, want 4", got)
	}
	if _, ok := data.MapGenSettings["autoplace_controls"]; !ok {
		t.Fatal("expected autoplace_controls in map gen settings")
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
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed reading working directory: %v", err)
	}
	t.Cleanup(func() {
		cfg.Global.Paths.Folders.ServersRoot = oldRoot
		cfg.Global.Paths.Folders.MapGenerators = oldMapGenerators
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed restoring working directory: %v", err)
		}
	})

	tmp := t.TempDir()
	sharedRoot := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed changing working directory: %v", err)
	}
	cfg.Global.Paths.Folders.ServersRoot = sharedRoot + string(os.PathSeparator)
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

	sharedGen := filepath.Join(sharedRoot, "map-gen-json", constants.CustomMapGeneratorName+"-gen.json")
	if _, err := os.Stat(sharedGen); !os.IsNotExist(err) {
		t.Fatalf("custom map exchange file should not be written to shared generator folder: %s", sharedGen)
	}
}
