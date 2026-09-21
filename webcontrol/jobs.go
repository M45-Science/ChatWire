package webcontrol

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type UnknownOutcomeError struct{ Err error }

func (e UnknownOutcomeError) Error() string { return e.Err.Error() }
func (e UnknownOutcomeError) Unwrap() error { return e.Err }

type Job struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	Actor     Actor     `json:"actor"`
	Action    string    `json:"action"`
	State     string    `json:"state"`
	Phase     string    `json:"phase"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Result    any       `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	Key       string    `json:"-"`
	Hash      string    `json:"-"`
}
type jobDisk struct {
	Job  Job    `json:"job"`
	Key  string `json:"key"`
	Hash string `json:"hash"`
}
type Jobs struct {
	mu    sync.Mutex
	dir   string
	items map[string]Job
	keys  map[string]string
	busy  string
}

func NewJobs(dir string) (*Jobs, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	j := &Jobs{dir: dir, items: map[string]Job{}, keys: map[string]string{}}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".json" {
			continue
		}
		b, e := os.ReadFile(filepath.Join(dir, f.Name()))
		if e != nil {
			return nil, e
		}
		var d jobDisk
		if e = json.Unmarshal(b, &d); e != nil {
			return nil, e
		}
		v := d.Job
		v.Key = d.Key
		v.Hash = d.Hash
		if !ValidID(v.ID) {
			return nil, errors.New("invalid persisted job ID")
		}
		if v.State == "queued" || v.State == "running" || v.State == "waiting_for_empty" {
			v.State = "unknown"
			v.Error = "Instance restarted; operation was not replayed."
			v.UpdatedAt = time.Now().UTC()
			if e = j.save(v); e != nil {
				return nil, e
			}
		}
		if v.State == "waiting_for_restart" && v.Action == "chatwire-restart" {
			v.State = "succeeded"
			v.Phase = "reconnected"
			v.UpdatedAt = time.Now().UTC()
			v.Result = map[string]string{"status": "new ChatWire process started"}
			if e = j.save(v); e != nil {
				return nil, e
			}
		}
		j.items[v.ID] = v
		j.keys[v.Key] = v.ID
	}
	return j, nil
}
func (j *Jobs) save(v Job) error {
	b, e := json.Marshal(jobDisk{v, v.Key, v.Hash})
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(j.dir, ".job-")
	if e != nil {
		return e
	}
	p := f.Name()
	defer os.Remove(p)
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	if e = os.Rename(p, filepath.Join(j.dir, v.ID+".json")); e != nil {
		return e
	}
	d, e := os.Open(j.dir)
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}

// Start persists acceptance before calling run and deduplicates lost responses.
func (j *Jobs) Start(server string, a Actor, action, key string, payload []byte, run func(string) (any, error)) (Job, int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	k := Digest(a.ID + "\x00" + server + "\x00" + key)
	h := Digest(action + "\x00" + string(payload))
	if id, ok := j.keys[k]; ok {
		v := j.items[id]
		if v.Hash != h {
			return Job{}, 409, errors.New("idempotency key already used for a different request")
		}
		return v, 200, nil
	}
	if j.busy != "" {
		return Job{}, 409, errors.New("another control operation is active")
	}
	now := time.Now().UTC()
	v := Job{ID: "job_" + Token(), ServerID: server, Actor: a, Action: action, State: "queued", Phase: "accepted", CreatedAt: now, UpdatedAt: now, Key: k, Hash: h}
	if err := j.save(v); err != nil {
		return Job{}, 500, errors.New("cannot persist operation")
	}
	j.items[v.ID] = v
	j.keys[k] = v.ID
	j.busy = v.ID
	go func() {
		defer func() {
			if recover() != nil {
				_ = j.Update(v.ID, "unknown", "interrupted", nil, "Operation interrupted; inspect instance logs before retrying.")
			}
			j.mu.Lock()
			current := j.items[v.ID]
			if current.State == "succeeded" || current.State == "failed" || current.State == "unknown" || current.State == "interrupted" {
				j.busy = ""
			}
			j.mu.Unlock()
		}()
		if err := j.Update(v.ID, "running", "executing", nil, ""); err != nil {
			return
		}
		result, err := run(v.ID)
		state, msg := "succeeded", ""
		if err != nil {
			state = "failed"
			var unknown UnknownOutcomeError
			if errors.As(err, &unknown) {
				state = "unknown"
			}
			msg = err.Error()
		}
		_ = j.Update(v.ID, state, "complete", result, msg)
	}()
	return v, 202, nil
}
func (j *Jobs) Update(id, state, phase string, result any, msg string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	v, ok := j.items[id]
	if !ok {
		return errors.New("unknown job")
	}
	v.State = state
	v.Phase = phase
	v.Result = result
	v.Error = msg
	v.UpdatedAt = time.Now().UTC()
	if e := j.save(v); e != nil {
		return e
	}
	j.items[id] = v
	return nil
}
func (j *Jobs) Get(id string) (Job, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	v, ok := j.items[id]
	return v, ok
}
func (j *Jobs) List() []Job {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]Job, 0, len(j.items))
	for _, v := range j.items {
		out = append(out, v)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].CreatedAt.After(out[b].CreatedAt) })
	if len(out) > 100 {
		out = out[:100]
	}
	return out
}

// Lookup permits a lost response to be reconciled even after preview expiry.
func (j *Jobs) Lookup(server string, a Actor, action, key string, payload []byte) (Job, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	k := Digest(a.ID + "\x00" + server + "\x00" + key)
	id, ok := j.keys[k]
	if !ok {
		return Job{}, false, nil
	}
	v := j.items[id]
	if v.Hash != Digest(action+"\x00"+string(payload)) {
		return Job{}, true, errors.New("idempotency key already used for a different request")
	}
	return v, true, nil
}
