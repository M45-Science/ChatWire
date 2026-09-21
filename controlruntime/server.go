// Package controlruntime adapts an existing ChatWire process to the private API.
package controlruntime

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/webcontrol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Runtime struct {
	Config   webcontrol.InstanceConfig
	Jobs     *webcontrol.Jobs
	boot     string
	guild    string
	primary  bool
	mu       sync.Mutex
	previews map[string]preview
	uploads  map[string]Upload
	server   *http.Server
	listener net.Listener
}
type preview struct {
	Actor, Action, Hash, Revision string
	Expires                       time.Time
}
type Params struct {
	Reason            string   `json:"reason,omitempty"`
	WhenEmpty         bool     `json:"when_empty,omitempty"`
	Force             bool     `json:"force,omitempty"`
	SaveID            string   `json:"save_id,omitempty"`
	ExchangeString    string   `json:"exchange_string,omitempty"`
	Command           string   `json:"command,omitempty"`
	Operation         string   `json:"operation,omitempty"`
	Mods              []string `json:"mods,omitempty"`
	Version           string   `json:"version,omitempty"`
	Player            string   `json:"player,omitempty"`
	Level             int      `json:"level,omitempty"`
	IP                string   `json:"ip,omitempty"`
	UploadIDs         []string `json:"upload_ids,omitempty"`
	Start             bool     `json:"start,omitempty"`
	ConfirmationToken string   `json:"confirmation_token,omitempty"`
}

func New(c webcontrol.InstanceConfig) (*Runtime, error) {
	if !webcontrol.ValidID(c.ID) || !filepath.IsAbs(c.Socket) || !filepath.IsAbs(c.StateDir) {
		return nil, errors.New("instance ID, absolute socket and state directory required")
	}
	if e := os.MkdirAll(c.StateDir, 0700); e != nil {
		return nil, e
	}
	identityPath := filepath.Join(c.StateDir, "instance-id")
	identity, e := os.OpenFile(identityPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e == nil {
		_, e = identity.WriteString(c.ID)
		if e == nil {
			e = identity.Sync()
		}
		identity.Close()
		if e != nil {
			return nil, e
		}
	} else if os.IsExist(e) {
		b, err := os.ReadFile(identityPath)
		if err != nil || string(b) != c.ID {
			return nil, errors.New("state directory belongs to a different instance")
		}
	} else {
		return nil, e
	}
	jobs, e := webcontrol.NewJobs(filepath.Join(c.StateDir, "jobs"))
	if e != nil {
		return nil, e
	}
	rt := &Runtime{Config: c, Jobs: jobs, boot: webcontrol.Token(), guild: cfg.Global.Discord.Guild, primary: strings.EqualFold(cfg.Local.Callsign, cfg.Global.PrimaryServer), previews: map[string]preview{}, uploads: map[string]Upload{}}
	if e = os.MkdirAll(filepath.Join(c.StateDir, "uploads"), 0700); e != nil {
		return nil, e
	}
	entries, _ := os.ReadDir(filepath.Join(c.StateDir, "uploads"))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "upload-") {
			_ = os.Remove(filepath.Join(c.StateDir, "uploads", entry.Name()))
		}
	}
	return rt, nil
}
func Start(c webcontrol.InstanceConfig) (*Runtime, error) {
	secret, e := webcontrol.Credential(c.CredentialFile)
	if e != nil {
		return nil, e
	}
	l, e := webcontrol.Listen(c.Socket)
	if e != nil {
		return nil, e
	}
	rt, e := New(c)
	if e != nil {
		l.Close()
		return nil, e
	}
	cfg.EnableControlLocks()
	rt.listener = l
	rt.server = &http.Server{Handler: webcontrol.Authenticate(secret, rt.Handler()), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	go func() {
		if e := rt.server.Serve(l); e != nil && e != http.ErrServerClosed {
			cwlog.DoLogCW("web control listener stopped")
		}
	}()
	return rt, nil
}
func (rt *Runtime) Close(ctx context.Context) error { return rt.server.Shutdown(ctx) }
func (rt *Runtime) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /internal/v1/members/{user}/authorization", rt.authorization)
	mux.HandleFunc("GET /internal/v1/status", rt.status)
	mux.HandleFunc("GET /internal/v1/settings/schema", func(w http.ResponseWriter, r *http.Request) { webcontrol.JSON(w, 200, cfg.WebSchema(false)) })
	mux.HandleFunc("GET /internal/v1/settings/global/schema", func(w http.ResponseWriter, r *http.Request) {
		if !rt.requireAdmin(w, r) {
			return
		}
		webcontrol.JSON(w, 200, cfg.WebSchema(true))
	})
	mux.HandleFunc("/internal/v1/settings", rt.settings)
	mux.HandleFunc("/internal/v1/settings/global", rt.settings)
	mux.HandleFunc("GET /internal/v1/actions", func(w http.ResponseWriter, r *http.Request) { webcontrol.JSON(w, 200, rt.actions()) })
	mux.HandleFunc("POST /internal/v1/actions/{action}/preview", rt.previewAction)
	mux.HandleFunc("POST /internal/v1/actions/{action}", rt.action)
	mux.HandleFunc("GET /internal/v1/jobs", func(w http.ResponseWriter, r *http.Request) { webcontrol.JSON(w, 200, rt.Jobs.List()) })
	mux.HandleFunc("GET /internal/v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, ok := rt.Jobs.Get(r.PathValue("id"))
		if !ok {
			webcontrol.Error(w, 404, "not_found", "Unknown job.")
			return
		}
		webcontrol.JSON(w, 200, v)
	})
	mux.HandleFunc("GET /internal/v1/audit", func(w http.ResponseWriter, r *http.Request) {
		out := []map[string]any{}
		for _, v := range rt.Jobs.List() {
			out = append(out, map[string]any{"job_id": v.ID, "actor": v.Actor, "action": v.Action, "state": v.State, "created_at": v.CreatedAt})
		}
		webcontrol.JSON(w, 200, out)
	})
	rt.fileRoutes(mux)
	rt.playerRoutes(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-ChatWire-Instance", rt.Config.ID)
		if strings.HasPrefix(r.URL.Path, "/internal/v1/members/") {
			mux.ServeHTTP(w, r)
			return
		}
		var a webcontrol.Actor
		if r.Header.Get("X-ChatWire-Instance") != rt.Config.ID || json.Unmarshal([]byte(r.Header.Get("X-ChatWire-Actor")), &a) != nil || !webcontrol.ValidID(a.ID) {
			webcontrol.Error(w, 403, "forbidden", "Verified actor and matching instance required.")
			return
		}
		mux.ServeHTTP(w, r)
	})
}
func actor(r *http.Request) webcontrol.Actor {
	var a webcontrol.Actor
	_ = json.Unmarshal([]byte(r.Header.Get("X-ChatWire-Actor")), &a)
	return a
}
func (rt *Runtime) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !actor(r).Admin || !rt.primary {
		webcontrol.Error(w, 403, "forbidden", "Primary instance and administrator access required.")
		return false
	}
	return true
}
func (rt *Runtime) authorization(w http.ResponseWriter, r *http.Request) {
	if !rt.primary || disc.DS == nil || !webcontrol.ValidID(r.PathValue("user")) {
		webcontrol.Error(w, 503, "unavailable", "Discord authorization unavailable.")
		return
	}
	member, e := disc.DS.GuildMember(rt.guild, r.PathValue("user"), discordgo.WithContext(r.Context()))
	if e != nil || member.User == nil {
		webcontrol.Error(w, 403, "forbidden", "Cannot verify guild membership.")
		return
	}
	webcontrol.JSON(w, 200, webcontrol.Authorization{UserID: member.User.ID, Name: member.User.Username, GuildID: rt.guild, Roles: member.Roles})
}
func (rt *Runtime) status(w http.ResponseWriter, r *http.Request) {
	state := fact.GetLifecycleState()
	version, mapPath := fact.WebRuntimeMetadata()
	mapName := ""
	if mapPath != "" {
		mapName = filepath.Base(mapPath)
	}
	local, _ := cfg.WebRead(false)
	global, _ := cfg.WebRead(true)
	webcontrol.JSON(w, 200, map[string]any{"id": rt.Config.ID, "boot_id": rt.boot, "version": version, "map": mapName, "configuration": map[string]any{"local_desired_revision": local.Revision, "local_runtime_revision": cfg.RuntimeConfigRevision(false), "global_desired_revision": global.Revision, "global_runtime_revision": cfg.RuntimeConfigRevision(true)}, "lifecycle": state, "players": fact.NumPlayersCurrent(), "autostart": fact.AutostartEnabled(), "updating": fact.UpdateInProgress(), "mod_operation": fact.ModOperationInProgress()})
}
func (rt *Runtime) settings(w http.ResponseWriter, r *http.Request) {
	global := strings.HasSuffix(r.URL.Path, "/global")
	if global && !rt.requireAdmin(w, r) {
		return
	}
	switch r.Method {
	case "GET":
		v, e := cfg.WebRead(global)
		if e != nil {
			webcontrol.Error(w, 500, "settings_unavailable", "Unable to read settings.")
			return
		}
		w.Header().Set("ETag", `"`+v.Revision+`"`)
		webcontrol.JSON(w, 200, v)
	case "PATCH":
		expected := strings.Trim(r.Header.Get("If-Match"), `"`)
		if expected == "" {
			webcontrol.Error(w, 428, "precondition_required", "If-Match revision required.")
			return
		}
		var patch map[string]json.RawMessage
		if webcontrol.Decode(w, r, &patch) != nil || len(patch) == 0 {
			webcontrol.Error(w, 400, "invalid_request", "Settings patch required.")
			return
		}
		if !glob.ControlLock.TryLock() {
			webcontrol.Error(w, 409, "busy", "A Discord or web operation is active.")
			return
		}
		defer glob.ControlLock.Unlock()
		v, e := cfg.WebPatch(global, actor(r).Admin, expected, patch)
		if e != nil {
			status := 422
			if errors.Is(e, cfg.ErrRevision) {
				status = 412
			}
			webcontrol.Error(w, status, "settings_rejected", e.Error())
			return
		}
		cwlog.DoLogAudit("WEB SETTINGS: actor=%s global=%t revision=%s", actor(r).ID, global, v.Revision)
		w.Header().Set("ETag", `"`+v.Revision+`"`)
		webcontrol.JSON(w, 200, v)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		webcontrol.Error(w, 405, "method_not_allowed", "Use GET or PATCH.")
	}
}

type actionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Primary     bool   `json:"primary"`
}

func (rt *Runtime) actions() []actionInfo {
	out := []actionInfo{{"factorio-start", "Start Factorio and enable autostart.", false}, {"factorio-stop", "Save and stop Factorio; disable autostart.", false}, {"factorio-restart", "Restart Factorio, optionally when empty.", false}, {"chatwire-restart", "Restart ChatWire, optionally when empty or forced.", false}, {"config-reload", "Reload persisted local and global settings.", false}, {"map-create", "Generate a new save while stopped.", false}, {"map-reset", "Archive/reset the current map and restart.", false}, {"map-load", "Load the selected save, stopping the current map.", false}, {"map-exchange", "Generate a save from an exchange string while stopped.", false}, {"map-archive", "Archive a selected save.", false}, {"mods-update", "Check and install mod updates.", false}, {"mods-sync", "Synchronize mods with the save while stopped.", false}, {"mods-edit", "Add, remove, enable, disable, or pin a mod version.", false}, {"mods-clear", "Delete all mods and mod settings while stopped.", false}, {"mods-clear-history", "Clear mod history and updater exclusions.", false}, {"factorio-update", "Check and install Factorio updates.", false}, {"factorio-install", "Install Factorio.", false}, {"rcon", "Run a Factorio console command.", false}, {"upload-apply", "Activate staged saves/mod-list/mod-settings.", false}}
	if rt.primary {
		out = append(out, actionInfo{"player-level", "Change a player level or ban status.", true}, actionInfo{"ip-ban", "Add a public IPv4 firewall deny rule.", true}, actionInfo{"ip-unban", "Remove a public IPv4 firewall deny rule.", true})
	}
	return out
}
func (rt *Runtime) known(action string) bool {
	for _, v := range rt.actions() {
		if v.Name == action {
			return true
		}
	}
	return false
}
func (rt *Runtime) stateRevision() string {
	st := fact.GetLifecycleState()
	b, _ := json.Marshal(struct {
		State   fact.State
		Players int
	}{st, fact.NumPlayersCurrent()})
	l, _ := cfg.WebRead(false)
	g, _ := cfg.WebRead(true)
	return webcontrol.Digest(string(b) + l.Revision + g.Revision)
}
func (rt *Runtime) previewAction(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	if !rt.known(action) {
		webcontrol.Error(w, 404, "not_found", "Unknown action.")
		return
	}
	var p Params
	if webcontrol.Decode(w, r, &p) != nil {
		webcontrol.Error(w, 400, "invalid_request", "Invalid action parameters.")
		return
	}
	if target := r.PathValue("player"); target != "" {
		if p.Player != "" && p.Player != target {
			webcontrol.Error(w, 422, "target_mismatch", "Player path and body must match.")
			return
		}
		p.Player = target
	}
	p.ConfirmationToken = ""
	b, _ := json.Marshal(p)
	token := webcontrol.Token()
	rev := rt.stateRevision()
	rt.mu.Lock()
	for k, v := range rt.previews {
		if time.Now().After(v.Expires) {
			delete(rt.previews, k)
		}
	}
	if len(rt.previews) > 1000 {
		rt.mu.Unlock()
		webcontrol.Error(w, 429, "rate_limited", "Too many previews.")
		return
	}
	rt.previews[webcontrol.Digest(token)] = preview{actor(r).ID, action, webcontrol.Digest(string(b)), rev, time.Now().Add(2 * time.Minute)}
	rt.mu.Unlock()
	webcontrol.JSON(w, 200, map[string]any{"confirmation_token": token, "server_id": rt.Config.ID, "action": action, "players": fact.NumPlayersCurrent(), "state_revision": rev, "expires_in": 120})
}
func (rt *Runtime) action(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	if !rt.known(action) {
		webcontrol.Error(w, 404, "not_found", "Unknown action.")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if !webcontrol.ValidID(key) {
		webcontrol.Error(w, 428, "idempotency_required", "A unique Idempotency-Key is required.")
		return
	}
	var p Params
	if webcontrol.Decode(w, r, &p) != nil {
		webcontrol.Error(w, 400, "invalid_request", "Invalid action parameters.")
		return
	}
	token := p.ConfirmationToken
	if target := r.PathValue("player"); target != "" {
		if p.Player != "" && p.Player != target {
			webcontrol.Error(w, 422, "target_mismatch", "Player path and body must match.")
			return
		}
		p.Player = target
	}
	p.ConfirmationToken = ""
	b, _ := json.Marshal(p)
	a := actor(r)
	if job, found, err := rt.Jobs.Lookup(rt.Config.ID, a, action, key, b); found {
		if err != nil {
			webcontrol.Error(w, 409, "idempotency_conflict", err.Error())
			return
		}
		w.Header().Set("Location", "/internal/v1/jobs/"+job.ID)
		webcontrol.JSON(w, 200, job)
		return
	}
	// The preview may be reused only for identical idempotent retries. Job storage
	// makes those retries return the original operation without executing it again.
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if job, found, err := rt.Jobs.Lookup(rt.Config.ID, a, action, key, b); found {
		if err != nil {
			webcontrol.Error(w, 409, "idempotency_conflict", err.Error())
			return
		}
		webcontrol.JSON(w, 200, job)
		return
	}
	pv, ok := rt.previews[webcontrol.Digest(token)]
	if !ok || pv.Actor != a.ID || pv.Action != action || pv.Hash != webcontrol.Digest(string(b)) || time.Now().After(pv.Expires) {
		webcontrol.Error(w, 409, "confirmation_required", "Preview this action and confirm its targets first.")
		return
	}
	if pv.Revision != rt.stateRevision() {
		webcontrol.Error(w, 409, "state_changed", "Server state changed; preview the action again.")
		return
	}
	job, status, e := rt.Jobs.Start(rt.Config.ID, a, action, key, b, func(id string) (any, error) {
		if !glob.ControlLock.TryLock() {
			return nil, errors.New("another Discord or web operation is active")
		}
		defer glob.ControlLock.Unlock()
		cwlog.DoLogAudit("WEB ACTION: actor=%s action=%s job=%s", a.ID, action, id)
		result, err := rt.execute(id, a, action, p)
		if err != nil {
			cwlog.DoLogCW("web operation failed job=%s action=%s", id, action)
		}
		cwlog.DoLogAudit("WEB RESULT: actor=%s action=%s job=%s success=%t", a.ID, action, id, err == nil)
		return result, err
	})
	if e != nil {
		webcontrol.Error(w, status, "action_rejected", e.Error())
		return
	}
	delete(rt.previews, webcontrol.Digest(token))
	w.Header().Set("Location", "/internal/v1/jobs/"+job.ID)
	webcontrol.JSON(w, status, job)
}
func fail(msg string) error { return fmt.Errorf("%s", msg) }
