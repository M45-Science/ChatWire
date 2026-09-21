package webapi

import (
	"ChatWire/webcontrol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type remote struct {
	endpoint webcontrol.Endpoint
	client   *http.Client
	secret   string
}
type Server struct {
	auditMu      sync.Mutex
	configPath   string
	jobs         *webcontrol.Jobs
	previews     map[string]batchPreview
	config       Config
	auth         *Auth
	remotes      map[string]remote
	brokerSecret string
	mu           sync.Mutex
	lastSeen     map[string]time.Time
}

func New(c Config) (*Server, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	secret, e := webcontrol.Credential(c.BrokerCredentialFile)
	if e != nil {
		return nil, e
	}
	s := &Server{config: c, brokerSecret: secret, remotes: map[string]remote{}, lastSeen: map[string]time.Time{}}
	for _, ep := range c.Instances {
		if !ep.Enabled {
			continue
		}
		key, e := webcontrol.Credential(ep.CredentialFile)
		if e != nil {
			return nil, e
		}
		s.remotes[ep.ID] = remote{ep, webcontrol.UnixClient(ep.Socket), key}
	}
	s.jobs, e = webcontrol.NewJobs(filepath.Join(c.StateDir, "jobs"))
	if e != nil {
		return nil, e
	}
	s.auth = newAuth(func(ctx context.Context, id string) (webcontrol.Actor, error) {
		r := s.remotes[c.Primary]
		var a webcontrol.Authorization
		if e := s.remoteJSON(ctx, r, webcontrol.Actor{}, "GET", "/members/"+url.PathEscape(id)+"/authorization", nil, "", &a); e != nil {
			return webcontrol.Actor{}, errors.New("cannot verify Discord membership")
		}
		if a.UserID != id {
			return webcontrol.Actor{}, errors.New("identity mismatch")
		}
		return c.actor(a)
	})
	return s, nil
}
func (s *Server) Broker() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/v1/login-grants", func(w http.ResponseWriter, r *http.Request) {
		var req webcontrol.GrantRequest
		if webcontrol.Decode(w, r, &req) != nil {
			webcontrol.Error(w, 400, "invalid_request", "Invalid grant request.")
			return
		}
		token, expiry, err := s.auth.issue(r.Context(), req)
		if err != nil {
			webcontrol.Error(w, 403, "login_denied", err.Error())
			return
		}
		s.audit("login_grant", req.UserID, "")
		webcontrol.JSON(w, 201, webcontrol.Grant{URL: s.config.PublicOrigin + "/login#token=" + token, ExpiresAt: expiry})
	})
	mux.HandleFunc("POST /internal/v1/revocations", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID string `json:"user_id"`
		}
		if webcontrol.Decode(w, r, &req) != nil || !webcontrol.ValidID(req.UserID) {
			webcontrol.Error(w, 400, "invalid_request", "Invalid user.")
			return
		}
		s.auth.revoke(req.UserID)
		s.audit("revoke_sessions", req.UserID, req.UserID)
		webcontrol.JSON(w, 200, map[string]bool{"revoked": true})
	})
	return webcontrol.Authenticate(s.brokerSecret, mux)
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.batchRoutes(mux)
	s.hostRoutes(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { webcontrol.JSON(w, 200, map[string]bool{"ok": true}) })
	mux.HandleFunc("GET /login", s.login)
	mux.HandleFunc("POST /auth/exchange", s.exchange)
	mux.HandleFunc("POST /auth/logout", s.protected(false, func(w http.ResponseWriter, r *http.Request, a session) {
		c, _ := r.Cookie(sessionCookie)
		s.auth.logout(c.Value)
		s.audit("logout", a.Actor.ID, "")
		cookie(w, sessionCookie, "", -1)
		webcontrol.JSON(w, 200, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("POST /auth/revocations", s.protected(true, func(w http.ResponseWriter, r *http.Request, a session) {
		var v struct {
			UserID string `json:"user_id"`
		}
		if webcontrol.Decode(w, r, &v) != nil || !webcontrol.ValidID(v.UserID) {
			webcontrol.Error(w, 400, "invalid_request", "Invalid user.")
			return
		}
		s.auth.revoke(v.UserID)
		s.audit("admin_revoke_sessions", a.Actor.ID, v.UserID)
		webcontrol.JSON(w, 200, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("POST /auth/activity", s.protected(false, func(w http.ResponseWriter, r *http.Request, a session) {
		webcontrol.JSON(w, 200, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("GET /api/v1/me", s.protected(false, func(w http.ResponseWriter, r *http.Request, a session) {
		webcontrol.JSON(w, 200, map[string]any{"actor": a.Actor, "csrf_token": a.CSRF})
	}))
	mux.HandleFunc("GET /api/v1/servers", s.protected(false, func(w http.ResponseWriter, r *http.Request, a session) {
		webcontrol.JSON(w, 200, s.statuses(r.Context(), a.Actor))
	}))
	mux.HandleFunc("/api/v1/servers/{server}/{rest...}", s.protected(false, s.proxy))
	mux.HandleFunc("GET /api/v1/servers/{server}", s.protected(false, s.proxy))
	mux.HandleFunc("/api/v1/settings/global", s.protected(true, s.shared))
	mux.HandleFunc("GET /api/v1/settings/global/schema", s.protected(true, s.shared))
	for _, p := range []string{"/api/v1/players", "/api/v1/players/{rest...}", "/api/v1/host/ip-bans", "/api/v1/host/actions/{rest...}"} {
		mux.HandleFunc(p, s.protected(false, s.shared))
	}
	mux.HandleFunc("GET /api/v1/jobs", s.protected(false, func(w http.ResponseWriter, r *http.Request, a session) { s.aggregate(w, r, a, "jobs") }))
	mux.HandleFunc("GET /api/v1/audit", s.protected(false, func(w http.ResponseWriter, r *http.Request, a session) { s.aggregate(w, r, a, "audit") }))
	mux.HandleFunc("GET /api/v1/events", s.protected(false, s.events))
	for _, path := range []string{"GET /{$}", "GET /app.js", "GET /login.js", "GET /style.css"} {
		mux.HandleFunc(path, s.static)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		mux.ServeHTTP(w, r)
	})
}
func (s *Server) sameOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == s.config.PublicOrigin
}
func (s *Server) protected(admin bool, fn func(http.ResponseWriter, *http.Request, session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, e := r.Cookie(sessionCookie)
		if e != nil {
			webcontrol.Error(w, 401, "unauthorized", "Sign in using /web in Discord.")
			return
		}
		a, e := s.auth.check(r.Context(), c.Value, r.Method != "GET" && r.Method != "HEAD")
		if e != nil {
			webcontrol.Error(w, 401, "unauthorized", e.Error())
			return
		}
		if admin && !a.Actor.Admin {
			webcontrol.Error(w, 403, "forbidden", "Administrator access required.")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if !s.sameOrigin(r) || !webcontrol.Equal(r.Header.Get("X-CSRF-Token"), a.CSRF) {
				webcontrol.Error(w, 403, "csrf", "Invalid request origin or CSRF token.")
				return
			}
		}
		fn(w, r, a)
	}
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	n, e := s.auth.landing()
	if e != nil {
		webcontrol.Error(w, 429, "rate_limited", "Try again shortly.")
		return
	}
	cookie(w, nonceCookie, n, 300)
	s.asset(w, r, "login.html")
}
func (s *Server) exchange(w http.ResponseWriter, r *http.Request) {
	if !s.sameOrigin(r) {
		webcontrol.Error(w, 403, "csrf", "Invalid request origin.")
		return
	}
	if c, e := r.Cookie(sessionCookie); e == nil {
		if _, e = s.auth.check(r.Context(), c.Value, false); e == nil {
			webcontrol.Error(w, 409, "already_signed_in", "Already signed in. Open the dashboard or sign out first.")
			return
		}
	}
	var req struct {
		Token string `json:"token"`
	}
	c, e := r.Cookie(nonceCookie)
	if e != nil || webcontrol.Decode(w, r, &req) != nil {
		webcontrol.Error(w, 400, "invalid_login", "Invalid login request.")
		return
	}
	raw, signedIn, e := s.auth.exchange(r.Context(), req.Token, c.Value)
	if e != nil {
		webcontrol.Error(w, 401, "invalid_login", e.Error())
		return
	}
	s.audit("login_exchange", signedIn.Actor.ID, "")
	cookie(w, nonceCookie, "", -1)
	cookie(w, sessionCookie, raw, 28800)
	webcontrol.JSON(w, 200, map[string]string{"redirect": "/"})
}
func (s *Server) request(ctx context.Context, rm remote, method, path string, body io.Reader, actor webcontrol.Actor, h http.Header) (*http.Response, error) {
	req, e := http.NewRequestWithContext(ctx, method, "http://local/internal/v1"+path, body)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+rm.secret)
	req.Header.Set("X-ChatWire-Instance", rm.endpoint.ID)
	b, _ := json.Marshal(actor)
	req.Header.Set("X-ChatWire-Actor", string(b))
	for _, k := range []string{"Content-Type", "If-Match", "Idempotency-Key"} {
		req.Header.Set(k, h.Get(k))
	}
	client := rm.client
	if strings.HasPrefix(path, "/uploads") || strings.Contains(path, "/download") {
		copy := *client
		copy.Timeout = 10 * time.Minute
		client = &copy
	}
	return client.Do(req)
}
func (s *Server) proxy(w http.ResponseWriter, r *http.Request, a session) {
	id := r.PathValue("server")
	rm, ok := s.remotes[id]
	if !ok {
		webcontrol.Error(w, 404, "not_found", "Unknown server.")
		return
	}
	path := "/" + r.PathValue("rest")
	if path == "/" {
		path = "/status"
	}
	s.forward(w, r, a, rm, path)
}
func (s *Server) shared(w http.ResponseWriter, r *http.Request, a session) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	if path == "/players" {
		q := r.URL.Query()
		q.Set("scope", "all")
		r.URL.RawQuery = q.Encode()
	}
	s.forward(w, r, a, s.remotes[s.config.Primary], path)
}
func (s *Server) forward(w http.ResponseWriter, r *http.Request, a session, rm remote, path string) {
	if strings.Contains(path, "..") || strings.Contains(path, "\\") || strings.Contains(path, "%") {
		webcontrol.Error(w, 400, "invalid_path", "Invalid resource path.")
		return
	}
	limit := int64(1 << 20)
	if path == "/uploads" {
		limit = 512 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	resp, e := s.request(r.Context(), rm, r.Method, path+"?"+r.URL.RawQuery, r.Body, a.Actor, r.Header)
	if e != nil {
		webcontrol.Error(w, 503, "instance_unavailable", "Instance unavailable; check operation status before retrying.")
		return
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-ChatWire-Instance") != rm.endpoint.ID {
		webcontrol.Error(w, 502, "identity_mismatch", "Instance identity mismatch.")
		return
	}
	for _, k := range []string{"Content-Type", "ETag", "Content-Disposition"} {
		if v := resp.Header.Get(k); v != "" {
			w.Header().Set(k, v)
		}
	}
	if loc := resp.Header.Get("Location"); strings.HasPrefix(loc, "/internal/v1/jobs/") {
		w.Header().Set("Location", "/api/v1/servers/"+rm.endpoint.ID+strings.TrimPrefix(loc, "/internal/v1"))
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// eachRemote bounds fan-out and preserves registry order in overview responses.
func (s *Server) eachRemote(fn func(int, remote)) {
	remotes := []remote{}
	for _, ep := range s.config.Instances {
		if rm, ok := s.remotes[ep.ID]; ok {
			remotes = append(remotes, rm)
		}
	}
	var wg sync.WaitGroup
	limit := make(chan struct{}, 8)
	for i, rm := range remotes {
		wg.Add(1)
		go func() { defer wg.Done(); limit <- struct{}{}; defer func() { <-limit }(); fn(i, rm) }()
	}
	wg.Wait()
}
func (s *Server) statuses(ctx context.Context, a webcontrol.Actor) []map[string]any {
	out := make([]map[string]any, len(s.remotes))
	s.eachRemote(func(i int, rm remote) {
		ep := rm.endpoint
		row := map[string]any{"id": ep.ID, "label": ep.Label, "available": false}
		ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		resp, e := s.request(ctx2, rm, "GET", "/status", nil, a, http.Header{})
		if e == nil {
			var v any
			if resp.StatusCode == 200 && resp.Header.Get("X-ChatWire-Instance") == ep.ID && json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&v) == nil {
				row["available"] = true
				row["status"] = v
				s.mu.Lock()
				s.lastSeen[ep.ID] = time.Now().UTC()
				s.mu.Unlock()
			}
			resp.Body.Close()
		}
		s.mu.Lock()
		row["last_seen"] = s.lastSeen[ep.ID]
		s.mu.Unlock()
		out[i] = row
	})
	return out
}
func (s *Server) aggregate(w http.ResponseWriter, r *http.Request, a session, resource string) {
	out := map[string]any{}
	var mu sync.Mutex
	if resource == "audit" {
		out["host"] = s.auditEntries()
	}
	if resource == "jobs" && s.jobs != nil {
		out["host"] = s.jobs.List()
	}
	s.eachRemote(func(_ int, rm remote) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		var result any = map[string]string{"error": "unavailable"}
		resp, e := s.request(ctx, rm, "GET", "/"+resource, nil, a.Actor, http.Header{})
		if e == nil {
			var v any
			if resp.StatusCode == 200 && resp.Header.Get("X-ChatWire-Instance") == rm.endpoint.ID && json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&v) == nil {
				result = v
			}
			resp.Body.Close()
		}
		mu.Lock()
		out[rm.endpoint.ID] = result
		mu.Unlock()
	})
	webcontrol.JSON(w, 200, out)
}
func (s *Server) events(w http.ResponseWriter, r *http.Request, a session) {
	f, ok := w.(http.Flusher)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, "event: resync_required\ndata: {}\n\n")
	f.Flush()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	c, _ := r.Cookie(sessionCookie)
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			if _, e := s.auth.check(r.Context(), c.Value, false); e != nil {
				return
			}
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			fmt.Fprint(w, "event: resync_required\ndata: {}\n\n")
			f.Flush()
		}
	}
}
