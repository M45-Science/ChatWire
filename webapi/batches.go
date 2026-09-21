package webapi

import (
	"ChatWire/webcontrol"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

type batchInput struct {
	Action            string          `json:"action"`
	Servers           []string        `json:"servers"`
	Parameters        json.RawMessage `json:"parameters"`
	ConfirmationToken string          `json:"confirmation_token,omitempty"`
}
type batchPreview struct {
	Actor, Hash string
	Expires     time.Time
	Children    map[string]string
}

func (s *Server) batchRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/batches/preview", s.protected(false, s.batchPreview))
	mux.HandleFunc("POST /api/v1/batches", s.protected(false, s.batch))
	mux.HandleFunc("GET /api/v1/jobs/{job}", s.protected(false, func(w http.ResponseWriter, r *http.Request, a session) {
		if s.jobs == nil {
			webcontrol.Error(w, 503, "unavailable", "Job store unavailable.")
			return
		}
		v, ok := s.jobs.Get(r.PathValue("job"))
		if !ok {
			webcontrol.Error(w, 404, "not_found", "Unknown job.")
			return
		}
		webcontrol.JSON(w, 200, v)
	}))
}
func (s *Server) normalizeBatch(w http.ResponseWriter, r *http.Request) (batchInput, []byte, bool) {
	var p batchInput
	if webcontrol.Decode(w, r, &p) != nil || len(p.Servers) == 0 || len(p.Servers) > 64 || (p.Action != "rcon" && p.Action != "config-reload") {
		webcontrol.Error(w, 422, "invalid_batch", "Choose explicit servers and rcon or config-reload.")
		return p, nil, false
	}
	if len(p.Parameters) == 0 {
		p.Parameters = json.RawMessage(`{}`)
	}
	var params map[string]any
	if e := json.Unmarshal(p.Parameters, &params); e != nil || params == nil {
		webcontrol.Error(w, 422, "invalid_batch", "parameters must be a JSON object.")
		return p, nil, false
	}
	p.Parameters, _ = json.Marshal(params)
	seen := map[string]bool{}
	for _, id := range p.Servers {
		if _, ok := s.remotes[id]; !ok || seen[id] {
			webcontrol.Error(w, 422, "invalid_batch", "Unknown or duplicate target server.")
			return p, nil, false
		}
		seen[id] = true
	}
	sort.Strings(p.Servers)
	copy := p
	copy.ConfirmationToken = ""
	b, _ := json.Marshal(copy)
	return p, b, true
}
func (s *Server) remoteJSON(ctx context.Context, rm remote, a webcontrol.Actor, method, path string, body any, key string, out any) error {
	b, e := json.Marshal(body)
	if e != nil {
		return e
	}
	h := http.Header{"Content-Type": []string{"application/json"}, "Idempotency-Key": []string{key}}
	resp, e := s.request(ctx, rm, method, path, bytes.NewReader(b), a, h)
	if e != nil {
		return errors.New("instance unavailable")
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-ChatWire-Instance") != rm.endpoint.ID {
		return errors.New("instance identity mismatch")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("instance rejected request; refresh state and preview again")
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
	}
	return nil
}
func (s *Server) batchPreview(w http.ResponseWriter, r *http.Request, a session) {
	p, b, ok := s.normalizeBatch(w, r)
	if !ok {
		return
	}
	children := map[string]string{}
	for _, id := range p.Servers {
		var out struct {
			Token string `json:"confirmation_token"`
		}
		if e := s.remoteJSON(r.Context(), s.remotes[id], a.Actor, "POST", "/actions/"+p.Action+"/preview", p.Parameters, "", &out); e != nil {
			webcontrol.Error(w, 503, "preview_failed", id+": "+e.Error())
			return
		}
		children[id] = out.Token
	}
	token := webcontrol.Token()
	s.mu.Lock()
	if s.previews == nil {
		s.previews = map[string]batchPreview{}
	}
	for k, v := range s.previews {
		if time.Now().After(v.Expires) {
			delete(s.previews, k)
		}
	}
	if len(s.previews) >= 1000 {
		s.mu.Unlock()
		webcontrol.Error(w, 429, "rate_limited", "Too many previews.")
		return
	}
	s.previews[webcontrol.Digest(token)] = batchPreview{a.Actor.ID, webcontrol.Digest(string(b)), time.Now().Add(2 * time.Minute), children}
	s.mu.Unlock()
	webcontrol.JSON(w, 200, map[string]any{"confirmation_token": token, "servers": p.Servers, "action": p.Action})
}
func (s *Server) batch(w http.ResponseWriter, r *http.Request, a session) {
	p, b, ok := s.normalizeBatch(w, r)
	if !ok {
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if !webcontrol.ValidID(key) {
		webcontrol.Error(w, 428, "idempotency_required", "Idempotency-Key required.")
		return
	}
	if s.jobs == nil {
		webcontrol.Error(w, 503, "unavailable", "Job store unavailable.")
		return
	}
	if v, found, e := s.jobs.Lookup("host", a.Actor, "batch-"+p.Action, key, b); found {
		if e != nil {
			webcontrol.Error(w, 409, "conflict", e.Error())
			return
		}
		webcontrol.JSON(w, 200, v)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, found, e := s.jobs.Lookup("host", a.Actor, "batch-"+p.Action, key, b); found {
		if e != nil {
			webcontrol.Error(w, 409, "conflict", e.Error())
			return
		}
		webcontrol.JSON(w, 200, v)
		return
	}
	preview, found := s.previews[webcontrol.Digest(p.ConfirmationToken)]
	if !found || preview.Actor != a.Actor.ID || preview.Hash != webcontrol.Digest(string(b)) || time.Now().After(preview.Expires) {
		webcontrol.Error(w, 409, "confirmation_required", "Preview the exact target list first.")
		return
	}
	v, status, e := s.jobs.Start("host", a.Actor, "batch-"+p.Action, key, b, func(parent string) (any, error) {
		results := map[string]any{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 4)
		failed := false
		for _, id := range p.Servers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				var params map[string]any
				_ = json.Unmarshal(p.Parameters, &params)
				params["confirmation_token"] = preview.Children[id]
				var job webcontrol.Job
				ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
				defer cancel()
				err := s.remoteJSON(ctx, s.remotes[id], a.Actor, "POST", "/actions/"+p.Action, params, webcontrol.Digest(parent+id), &job)
				if err == nil {
					for job.State == "queued" || job.State == "running" || job.State == "waiting_for_empty" || job.State == "waiting_for_restart" {
						select {
						case <-ctx.Done():
							err = ctx.Err()
						case <-time.After(time.Second):
							err = s.remoteJSON(ctx, s.remotes[id], a.Actor, "GET", "/jobs/"+job.ID, nil, "", &job)
						}
						if err != nil {
							break
						}
					}
				}
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					failed = true
					results[id] = map[string]string{"state": "unknown", "message": "Response unavailable; inspect instance jobs before retrying."}
				} else {
					results[id] = job
					if job.State != "succeeded" {
						failed = true
					}
				}
			}()
		}
		wg.Wait()
		if failed {
			return results, errors.New("One or more targets failed or have an unknown outcome.")
		}
		return results, nil
	})
	if e != nil {
		webcontrol.Error(w, status, "batch_rejected", e.Error())
		return
	}
	delete(s.previews, webcontrol.Digest(p.ConfirmationToken))
	w.Header().Set("Location", "/api/v1/jobs/"+v.ID)
	webcontrol.JSON(w, status, v)
}
