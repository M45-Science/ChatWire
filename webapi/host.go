package webapi

import (
	"ChatWire/webcontrol"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SetConfigPath enables admin edits to the persisted host registry. Changes are
// deliberately pending until the operator restarts the web service.
func (s *Server) SetConfigPath(path string) { s.configPath = path }
func (s *Server) hostRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/host/settings", s.protected(true, func(w http.ResponseWriter, r *http.Request, a session) {
		s.mu.Lock()
		defer s.mu.Unlock()
		v := s.config
		if s.configPath != "" {
			if e := webcontrol.LoadConfig(s.configPath, &v); e != nil {
				webcontrol.Error(w, 500, "storage_error", "Cannot read host settings.")
				return
			}
		}
		b, _ := json.Marshal(v)
		rev := webcontrol.Digest(string(b))
		w.Header().Set("ETag", `"`+rev+`"`)
		webcontrol.JSON(w, 200, map[string]any{"values": v, "revision": rev, "effect": "web_restart"})
	}))
	mux.HandleFunc("PATCH /api/v1/host/settings", s.protected(true, func(w http.ResponseWriter, r *http.Request, a session) {
		if s.configPath == "" {
			webcontrol.Error(w, 503, "unavailable", "Host registry path is not configured.")
			return
		}
		var v Config
		if webcontrol.Decode(w, r, &v) != nil || v.Validate() != nil {
			webcontrol.Error(w, 422, "invalid_config", "Invalid host registry; check listener, origin, primary, role IDs and paths.")
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		var old Config
		if e := webcontrol.LoadConfig(s.configPath, &old); e != nil {
			webcontrol.Error(w, 500, "storage_error", "Cannot read host settings.")
			return
		}
		b, _ := json.Marshal(old)
		etag := r.Header.Get("If-Match")
		if etag == "" {
			webcontrol.Error(w, 428, "precondition_required", "If-Match required.")
			return
		}
		if strings.Trim(etag, `"`) != webcontrol.Digest(string(b)) {
			webcontrol.Error(w, 412, "revision_conflict", "Host settings changed; refresh before saving.")
			return
		}
		data, _ := json.MarshalIndent(v, "", "  ")
		f, e := os.CreateTemp(filepath.Dir(s.configPath), ".web-config-")
		if e != nil {
			webcontrol.Error(w, 500, "storage_error", "Cannot save host settings.")
			return
		}
		defer os.Remove(f.Name())
		_, e = f.Write(data)
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e == nil {
			e = closeErr
		}
		if e == nil {
			e = os.Rename(f.Name(), s.configPath)
		}
		if e != nil {
			webcontrol.Error(w, 500, "storage_error", "Cannot save host settings.")
			return
		}
		s.audit("host_settings_saved", a.Actor.ID, "")
		webcontrol.JSON(w, 200, map[string]string{"status": "saved", "effect": "web_restart"})
	}))
}
