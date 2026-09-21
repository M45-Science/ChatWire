package controlruntime

import (
	"ChatWire/webcontrol"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateIdentityAndSharedPermission(t *testing.T) {
	dir := t.TempDir()
	rt, e := New(webcontrol.InstanceConfig{ID: "a", Socket: filepath.Join(dir, "a.sock"), StateDir: filepath.Join(dir, "state")})
	if e != nil {
		t.Fatal(e)
	}
	rt.primary = true
	handler := webcontrol.Authenticate("secret", rt.Handler())
	for _, tc := range []struct {
		path, key, id string
		admin         bool
		want          int
	}{{"/internal/v1/status", "", "a", false, 401}, {"/internal/v1/status", "secret", "b", false, 403}, {"/internal/v1/status", "secret", "a", false, 200}, {"/internal/v1/settings/global/schema", "secret", "a", false, 403}, {"/internal/v1/settings/global/schema", "secret", "a", true, 200}} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Authorization", "Bearer "+tc.key)
		r.Header.Set("X-ChatWire-Instance", tc.id)
		a, _ := json.Marshal(webcontrol.Actor{ID: "123", Admin: tc.admin})
		r.Header.Set("X-ChatWire-Actor", string(a))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s got %d want %d: %s", tc.path, w.Code, tc.want, w.Body.String())
		}
	}
}
func TestKnownJobRetryDoesNotNeedFreshPreview(t *testing.T) {
	dir := t.TempDir()
	rt, e := New(webcontrol.InstanceConfig{ID: "a", Socket: filepath.Join(dir, "a.sock"), StateDir: filepath.Join(dir, "state")})
	if e != nil {
		t.Fatal(e)
	}
	a := webcontrol.Actor{ID: "123"}
	p := Params{Command: "help"}
	b, _ := json.Marshal(p)
	release := make(chan struct{})
	defer close(release)
	job, _, e := rt.Jobs.Start("a", a, "rcon", "retry-key", b, func(string) (any, error) { <-release; return nil, nil })
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("POST", "/internal/v1/actions/rcon", strings.NewReader(`{"command":"help"}`))
	r.Header.Set("X-ChatWire-Instance", "a")
	actorBytes, _ := json.Marshal(a)
	r.Header.Set("X-ChatWire-Actor", string(actorBytes))
	r.Header.Set("Idempotency-Key", "retry-key")
	w := httptest.NewRecorder()
	rt.Handler().ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), job.ID) {
		t.Fatalf("retry failed: %d %s", w.Code, w.Body.String())
	}
}
