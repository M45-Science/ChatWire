package util

import (
	"bytes"
	"encoding/json"
	"os"
)

// WriteJSONAtomic writes data to path as JSON using a temporary file.
func WriteJSONAtomic(path string, data interface{}, perm os.FileMode) error {
	tempPath := path + ".tmp"

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")
	if err := enc.Encode(data); err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, outbuf.Bytes(), perm); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
