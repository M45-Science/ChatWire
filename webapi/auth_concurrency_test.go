package webapi

import (
	"ChatWire/webcontrol"
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type slowAuthRequest struct{}

func TestPassiveRequestPreservesConcurrentActivity(t *testing.T) {
	initial := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	var clock atomic.Int64
	clock.Store(initial.Add(29 * time.Minute).UnixNano())
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	actor := webcontrol.Actor{ID: "123", Name: "Moderator"}
	a := newAuth(func(ctx context.Context, _ string) (webcontrol.Actor, error) {
		if ctx.Value(slowAuthRequest{}) == true {
			close(entered)
			<-release
		}
		return actor, nil
	})
	a.now = func() time.Time { return time.Unix(0, clock.Load()) }
	raw := "test-session"
	a.sessions[webcontrol.Digest(raw)] = session{Actor: actor, Created: initial, Seen: initial, Checked: initial}
	go func() {
		_, err := a.check(context.WithValue(context.Background(), slowAuthRequest{}, true), raw, false)
		done <- err
	}()
	<-entered
	if _, err := a.check(context.Background(), raw, true); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	clock.Store(initial.Add(31 * time.Minute).UnixNano())
	if _, err := a.check(context.Background(), raw, false); err != nil {
		t.Fatalf("session expired only 2 minutes after activity: %v", err)
	}
}

func TestSlowRoleRefreshPreservesNewerAuthorization(t *testing.T) {
	initial := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	var clock atomic.Int64
	clock.Store(initial.Add(time.Minute).UnixNano())
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan session, 1)
	actor := webcontrol.Actor{ID: "123", Admin: true}
	a := newAuth(func(ctx context.Context, _ string) (webcontrol.Actor, error) {
		if ctx.Value(slowAuthRequest{}) == true {
			close(entered)
			<-release
			return actor, nil
		}
		return webcontrol.Actor{ID: "123", Admin: false}, nil
	})
	a.now = func() time.Time { return time.Unix(0, clock.Load()) }
	raw := "test-session"
	a.sessions[webcontrol.Digest(raw)] = session{Actor: actor, Created: initial, Seen: initial, Checked: initial}
	go func() {
		s, _ := a.check(context.WithValue(context.Background(), slowAuthRequest{}, true), raw, false)
		done <- s
	}()
	<-entered
	clock.Store(initial.Add(2 * time.Minute).UnixNano())
	current, err := a.check(context.Background(), raw, true)
	close(release)
	if err != nil {
		t.Fatal(err)
	}
	stale := <-done
	if stale.Actor.Admin || !stale.Checked.Equal(current.Checked) || !stale.Seen.Equal(current.Seen) {
		t.Fatalf("slow refresh overwrote newer session: current=%+v stale=%+v", current, stale)
	}
}
