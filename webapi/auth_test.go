package webapi

import (
	"ChatWire/webcontrol"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testAuth() *Auth {
	return newAuth(func(_ context.Context, id string) (webcontrol.Actor, error) {
		return webcontrol.Actor{ID: id, Name: "Moderator"}, nil
	})
}
func TestGrantConcurrentSingleUse(t *testing.T) {
	a := testAuth()
	token, _, err := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "456"})
	if err != nil {
		t.Fatal(err)
	}
	nonce, _ := a.landing()
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := a.exchange(context.Background(), token, nonce); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("redemptions=%d", successes.Load())
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.grants[token]; ok {
		t.Fatal("raw token stored")
	}
}
func TestGrantExpiryReplacementAndRevocation(t *testing.T) {
	a := testAuth()
	now := time.Now()
	a.now = func() time.Time { return now }
	first, _, _ := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "1"})
	now = now.Add(6 * time.Second)
	second, _, _ := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "2"})
	n, _ := a.landing()
	if _, _, e := a.exchange(context.Background(), first, n); e == nil {
		t.Fatal("old grant accepted")
	}
	a.revoke("123")
	if _, _, e := a.exchange(context.Background(), second, n); e == nil {
		t.Fatal("revoked grant accepted")
	}
	now = now.Add(6 * time.Second)
	third, _, _ := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "3"})
	now = now.Add(3 * time.Minute)
	if _, _, e := a.exchange(context.Background(), third, n); e == nil {
		t.Fatal("expired grant accepted")
	}
}
func TestSessionRoleRecheckAndIdleExpiry(t *testing.T) {
	a := testAuth()
	now := time.Now()
	a.now = func() time.Time { return now }
	token, _, _ := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "1"})
	n, _ := a.landing()
	raw, _, e := a.exchange(context.Background(), token, n)
	if e != nil {
		t.Fatal(e)
	}
	now = now.Add(61 * time.Second)
	a.authorize = func(context.Context, string) (webcontrol.Actor, error) {
		return webcontrol.Actor{}, errors.New("role removed")
	}
	if _, e = a.check(context.Background(), raw, true); e == nil {
		t.Fatal("removed role still authorized")
	}
	a = testAuth()
	a.now = func() time.Time { return now }
	token, _, _ = a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "2"})
	n, _ = a.landing()
	raw, _, _ = a.exchange(context.Background(), token, n)
	now = now.Add(29 * time.Minute)
	if _, e = a.check(context.Background(), raw, false); e != nil {
		t.Fatal(e)
	}
	now = now.Add(2 * time.Minute)
	if _, e = a.check(context.Background(), raw, false); e == nil {
		t.Fatal("passive traffic extended idle timeout")
	}
}
func TestPublicExchangeOriginAndPrefetch(t *testing.T) {
	a := testAuth()
	s := &Server{config: Config{PublicOrigin: "https://control.example"}, auth: a}
	h := s.Handler()
	token, _, _ := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "1"})
	// nginx terminates HTTPS; ChatWire receives plain HTTP and the browser Origin.
	req := httptest.NewRequest("GET", "http://127.0.0.1:8787/login", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var nonce *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == nonceCookie {
			nonce = c
		}
	}
	if nonce == nil || !nonce.Secure || !nonce.HttpOnly {
		t.Fatal("missing protected nonce cookie")
	}
	body := `{"token":"` + token + `"}`
	req = httptest.NewRequest("POST", "http://127.0.0.1:8787/auth/exchange", strings.NewReader(body))
	req.AddCookie(nonce)
	req.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1:8787/auth/exchange", strings.NewReader(body))
	req.AddCookie(nonce)
	req.Header.Set("Origin", "https://control.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("exchange=%d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), token) {
		t.Fatal("token leaked")
	}
	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			found = true
			if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
				t.Fatal("unprotected session")
			}
		}
	}
	if !found {
		t.Fatal("session not created")
	}
	req = httptest.NewRequest("POST", "https://control.example/internal/v1/login-grants", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code == 200 || w.Code == 201 {
		t.Fatal("public grant issuance")
	}
}
func TestCSRFAndAdminChecks(t *testing.T) {
	a := testAuth()
	token, _, _ := a.issue(context.Background(), webcontrol.GrantRequest{UserID: "123", InteractionID: "1"})
	n, _ := a.landing()
	raw, session, _ := a.exchange(context.Background(), token, n)
	s := &Server{config: Config{PublicOrigin: "https://control.example"}, auth: a}
	for _, tc := range []struct {
		path, csrf string
		want       int
	}{{"/auth/logout", "", 403}, {"/auth/revocations", session.CSRF, 403}} {
		r := httptest.NewRequest("POST", tc.path, strings.NewReader(`{"user_id":"123"}`))
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: raw})
		r.Header.Set("Origin", s.config.PublicOrigin)
		r.Header.Set("X-CSRF-Token", tc.csrf)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s=%d", tc.path, w.Code)
		}
	}
}

func TestBatchRejectsNonObjectParameters(t *testing.T) {
	s := &Server{remotes: map[string]remote{"a": {}}}
	for _, parameters := range []string{"null", "[]", "\"text\""} {
		r := httptest.NewRequest("POST", "/api/v1/batches", strings.NewReader(`{"action":"rcon","servers":["a"],"parameters":`+parameters+`}`))
		w := httptest.NewRecorder()
		if _, _, ok := s.normalizeBatch(w, r); ok || w.Code != 422 {
			t.Fatalf("accepted %s", parameters)
		}
	}
}
