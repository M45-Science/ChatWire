package util

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// CopyFileAtomic replaces dst only after the complete copy has been closed and
// synced. Preserve the source timestamp so backups do not become the newest save.
func CopyFileAtomic(src, dst string, perm os.FileMode) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()
	info, err := from.Stat()
	if err != nil {
		return err
	}
	to, err := os.CreateTemp(filepath.Dir(dst), ".cw-copy-*.tmp")
	if err != nil {
		return err
	}
	temp := to.Name()
	defer os.Remove(temp)
	defer to.Close()
	if err := to.Chmod(perm); err != nil {
		return err
	}
	if _, err := io.Copy(to, from); err != nil {
		return err
	}
	if err := to.Sync(); err != nil {
		return err
	}
	if err := to.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(temp, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	return os.Rename(temp, dst)
}

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

// WriteBytesAtomic writes raw bytes to path using a temporary file.
func WriteBytesAtomic(path string, data []byte, perm os.FileMode) error {
	WriteJSONLock.Lock()
	defer WriteJSONLock.Unlock()

	tmpName := TempFilePrefix + filepath.Base(path) + ".tmp"
	tempPath := filepath.Join(filepath.Dir(path), tmpName)

	if err := os.WriteFile(tempPath, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}
