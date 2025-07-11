package util

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// TempFilePrefix is applied to temporary files created by WriteJSONAtomic.
var TempFilePrefix string

// SetTempFilePrefix configures the prefix for atomic temp files.
func SetTempFilePrefix(prefix string) {
	TempFilePrefix = prefix
}

// WriteJSONAtomic writes data to path as JSON using a temporary file.
var WriteJSONLock sync.Mutex

func WriteJSONAtomic(path string, data interface{}, perm os.FileMode) error {
	WriteJSONLock.Lock()
	defer WriteJSONLock.Unlock()

	tmpName := TempFilePrefix + filepath.Base(path) + ".tmp"
	tempPath := filepath.Join(filepath.Dir(path), tmpName)

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")
	if err := enc.Encode(data); err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, outbuf.Bytes(), perm); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}
