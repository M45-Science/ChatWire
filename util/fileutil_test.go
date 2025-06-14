package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	SetTempFilePrefix("tmp-")
	data := map[string]int{"a": 1}
	if err := WriteJSONAtomic(path, data, 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var got map[string]int
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["a"] != 1 {
		t.Fatalf("unexpected data %v", got)
	}
	tmp := filepath.Join(dir, "tmp-data.json.tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temporary file not cleaned: %v", tmp)
	}
}
