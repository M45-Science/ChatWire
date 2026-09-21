package webapi

import (
	"ChatWire/webcontrol"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeInstance(t *testing.T, dir, id, reported string) webcontrol.Endpoint {
	t.Helper()
	socket := filepath.Join(dir, id+".sock")
	secretPath := filepath.Join(dir, id+".key")
	secret := "test-credential-01234567890123456789"
	if e := os.WriteFile(secretPath, []byte(secret), 0600); e != nil {
		t.Fatal(e)
	}
	l, e := net.Listen("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	server := &http.Server{Handler: webcontrol.Authenticate(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-ChatWire-Instance", reported)
		if r.URL.Path == "/internal/v1/members/123/authorization" {
			webcontrol.JSON(w, 200, webcontrol.Authorization{UserID: "123", GuildID: "guild", Roles: []string{"2"}})
			return
		}
		var a webcontrol.Actor
		if json.Unmarshal([]byte(r.Header.Get("X-ChatWire-Actor")), &a) != nil || a.ID != "123" {
			t.Errorf("verified actor not forwarded")
		}
		webcontrol.JSON(w, 200, map[string]string{"instance": id})
	}))}
	go server.Serve(l)
	t.Cleanup(func() { server.Close() })
	return webcontrol.Endpoint{ID: id, Label: id, Enabled: true, Socket: socket, CredentialFile: secretPath}
}
func TestGatewayRoutesToRegisteredSocketAndRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	a := fakeInstance(t, dir, "a", "a")
	b := fakeInstance(t, dir, "b", "wrong")
	c := Config{Listen: "127.0.0.1:8787", PublicOrigin: "https://control.example", StateDir: filepath.Join(dir, "state"), BrokerSocket: filepath.Join(dir, "broker.sock"), BrokerCredentialFile: a.CredentialFile, Primary: "a", GuildID: "guild", AdminRoleIDs: []string{"1"}, ModeratorRoleIDs: []string{"2"}, Instances: []webcontrol.Endpoint{a, b}}
	s, e := New(c)
	if e != nil {
		t.Fatal(e)
	}
	token, _, e := s.auth.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "456"})
	if e != nil {
		t.Fatal(e)
	}
	n, _ := s.auth.landing()
	raw, _, e := s.auth.exchange(context.Background(), token, n)
	if e != nil {
		t.Fatal(e)
	}
	h := s.Handler()
	for _, tc := range []struct {
		path string
		want int
	}{{"/api/v1/servers/a", 200}, {"/api/v1/servers/b", 502}, {"/api/v1/servers/unknown", 404}} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: raw})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s got %d: %s", tc.path, w.Code, w.Body.String())
		}
	}
	req := httptest.NewRequest("POST", "/internal/v1/login-grants", nil)
	w := httptest.NewRecorder()
	s.Broker().ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("broker accepted unauthenticated request")
	}
}
func TestSessionRevokedDuringRefreshCannotBeResurrected(t *testing.T) {
	a := testAuth()
	now := time.Now()
	a.now = func() time.Time { return now }
	token, _, _ := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "1"})
	n, _ := a.landing()
	raw, _, _ := a.exchange(context.Background(), token, n)
	now = now.Add(time.Minute + time.Second)
	entered := make(chan struct{})
	release := make(chan struct{})
	a.authorize = func(context.Context, string) (webcontrol.Actor, error) {
		close(entered)
		<-release
		return webcontrol.Actor{ID: "123"}, nil
	}
	done := make(chan error)
	go func() { _, e := a.check(context.Background(), raw, true); done <- e }()
	<-entered
	a.revoke("123")
	close(release)
	if e := <-done; e == nil {
		t.Fatal("revoked session resurrected")
	}
}
