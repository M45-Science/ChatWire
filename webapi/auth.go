package webapi

import (
	"ChatWire/webcontrol"
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"
)

const sessionCookie = "__Host-chatwire"
const nonceCookie = "__Host-chatwire-login"

type loginGrant struct {
	UserID  string
	Expires time.Time
}
type session struct {
	Actor                  webcontrol.Actor
	CSRF                   string
	Created, Seen, Checked time.Time
	Epoch                  uint64
}
type nonce struct{ Expires time.Time }
type Auth struct {
	mu           sync.Mutex
	grants       map[string]loginGrant
	sessions     map[string]session
	nonces       map[string]nonce
	interactions map[string]time.Time
	issued       map[string]time.Time
	epochs       map[string]uint64
	now          func() time.Time
	authorize    func(context.Context, string) (webcontrol.Actor, error)
}

func newAuth(authorize func(context.Context, string) (webcontrol.Actor, error)) *Auth {
	return &Auth{grants: map[string]loginGrant{}, sessions: map[string]session{}, nonces: map[string]nonce{}, interactions: map[string]time.Time{}, issued: map[string]time.Time{}, epochs: map[string]uint64{}, now: time.Now, authorize: authorize}
}
func (a *Auth) prune() {
	now := a.now()
	for k, v := range a.grants {
		if !now.Before(v.Expires) {
			delete(a.grants, k)
		}
	}
	for k, v := range a.nonces {
		if !now.Before(v.Expires) {
			delete(a.nonces, k)
		}
	}
	for k, v := range a.sessions {
		if now.Sub(v.Created) >= 8*time.Hour || now.Sub(v.Seen) >= 30*time.Minute {
			delete(a.sessions, k)
		}
	}
	for k, v := range a.interactions {
		if now.Sub(v) > 10*time.Minute {
			delete(a.interactions, k)
		}
	}
	for k, v := range a.issued {
		if now.Sub(v) > time.Minute {
			delete(a.issued, k)
		}
	}
}
func (a *Auth) issue(ctx context.Context, r webcontrol.GrantRequest) (string, time.Time, error) {
	if !webcontrol.ValidID(r.UserID) || !webcontrol.ValidID(r.InteractionID) {
		return "", time.Time{}, errors.New("invalid identity")
	}
	if _, e := a.authorize(ctx, r.UserID); e != nil {
		return "", time.Time{}, e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prune()
	now := a.now()
	if _, ok := a.interactions[r.InteractionID]; ok {
		return "", time.Time{}, errors.New("interaction already used; run /web again")
	}
	if last, ok := a.issued[r.UserID]; ok && now.Sub(last) < 5*time.Second {
		return "", time.Time{}, errors.New("wait a few seconds before requesting another link")
	}
	if len(a.grants) >= 1000 {
		return "", time.Time{}, errors.New("too many pending logins")
	}
	for k, v := range a.grants {
		if v.UserID == r.UserID {
			delete(a.grants, k)
		}
	}
	token := webcontrol.Token()
	expires := now.Add(2 * time.Minute)
	a.grants[webcontrol.Digest(token)] = loginGrant{r.UserID, expires}
	a.interactions[r.InteractionID] = now
	a.issued[r.UserID] = now
	return token, expires, nil
}
func (a *Auth) landing() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prune()
	if len(a.nonces) >= 10000 {
		return "", errors.New("too many login requests")
	}
	token := webcontrol.Token()
	a.nonces[webcontrol.Digest(token)] = nonce{a.now().Add(5 * time.Minute)}
	return token, nil
}
func (a *Auth) exchange(ctx context.Context, token, nonceToken string) (string, session, error) {
	digest := webcontrol.Digest(token)
	nd := webcontrol.Digest(nonceToken)
	a.mu.Lock()
	a.prune()
	g, ok := a.grants[digest]
	_, nok := a.nonces[nd]
	epoch := a.epochs[g.UserID]
	a.mu.Unlock()
	if !ok || !nok {
		return "", session{}, errors.New("invalid or expired login; run /web again")
	}
	actor, err := a.authorize(ctx, g.UserID)
	if err != nil {
		return "", session{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prune()
	if _, ok = a.grants[digest]; !ok {
		return "", session{}, errors.New("login link already used")
	}
	if _, ok = a.nonces[nd]; !ok || a.epochs[g.UserID] != epoch {
		return "", session{}, errors.New("login revoked")
	}
	delete(a.grants, digest)
	delete(a.nonces, nd)
	raw := webcontrol.Token()
	now := a.now()
	s := session{Actor: actor, CSRF: webcontrol.Token(), Created: now, Seen: now, Checked: now, Epoch: epoch}
	a.sessions[webcontrol.Digest(raw)] = s
	return raw, s, nil
}
func (a *Auth) check(ctx context.Context, raw string, touch bool) (session, error) {
	k := webcontrol.Digest(raw)
	a.mu.Lock()
	a.prune()
	s, ok := a.sessions[k]
	a.mu.Unlock()
	if !ok {
		return session{}, errors.New("sign in using /web in Discord")
	}
	if a.now().Sub(s.Checked) >= time.Minute {
		actor, e := a.authorize(ctx, s.Actor.ID)
		if e != nil {
			a.revoke(s.Actor.ID)
			return session{}, e
		}
		s.Actor = actor
		s.Checked = a.now()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prune()
	if _, ok = a.sessions[k]; !ok || a.epochs[s.Actor.ID] != s.Epoch {
		return session{}, errors.New("session revoked")
	}
	if touch {
		s.Seen = a.now()
	}
	a.sessions[k] = s
	return s, nil
}
func (a *Auth) revoke(user string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.epochs[user]++
	for k, v := range a.grants {
		if v.UserID == user {
			delete(a.grants, k)
		}
	}
	for k, v := range a.sessions {
		if v.Actor.ID == user {
			delete(a.sessions, k)
		}
	}
}
func (a *Auth) logout(raw string) {
	a.mu.Lock()
	delete(a.sessions, webcontrol.Digest(raw))
	a.mu.Unlock()
}
func cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (c Config) actor(v webcontrol.Authorization) (webcontrol.Actor, error) {
	if v.GuildID != c.GuildID || v.UserID == "" {
		return webcontrol.Actor{}, errors.New("guild membership required")
	}
	admin, mod := false, false
	for _, r := range v.Roles {
		admin = admin || slices.Contains(c.AdminRoleIDs, r)
		mod = mod || slices.Contains(c.ModeratorRoleIDs, r)
	}
	if !admin && !mod {
		return webcontrol.Actor{}, errors.New("moderator role required")
	}
	return webcontrol.Actor{ID: v.UserID, Name: v.Name, Admin: admin}, nil
}
