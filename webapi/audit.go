package webapi

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func (s *Server) audit(event, user, target string) {
	if s.config.StateDir == "" {
		return
	}
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	f, e := os.OpenFile(filepath.Join(s.config.StateDir, "audit.jsonl"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if e != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(map[string]any{"time": time.Now().UTC(), "event": event, "user_id": user, "target": target})
}
func (s *Server) auditEntries() []map[string]any {
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	out := []map[string]any{}
	f, e := os.Open(filepath.Join(s.config.StateDir, "audit.jsonl"))
	if e != nil {
		return out
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return out
	}
	offset := st.Size() - (1 << 20)
	if offset > 0 {
		_, _ = f.Seek(offset, 0)
	}
	scanner := bufio.NewScanner(f)
	if offset > 0 {
		scanner.Scan()
	}
	for scanner.Scan() {
		var row map[string]any
		if json.Unmarshal(scanner.Bytes(), &row) == nil {
			out = append(out, row)
			if len(out) > 100 {
				out = out[1:]
			}
		}
	}
	return out
}
