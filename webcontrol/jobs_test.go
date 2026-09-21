package webcontrol

import (
	"errors"
	"testing"
	"time"
)

func TestDurableIdempotencyAndRecovery(t *testing.T) {
	dir := t.TempDir()
	jobs, e := NewJobs(dir)
	if e != nil {
		t.Fatal(e)
	}
	release := make(chan struct{})
	done := make(chan struct{})
	a := Actor{ID: "123"}
	run := func(string) (any, error) { <-release; close(done); return "done", nil }
	first, status, e := jobs.Start("a", a, "restart", "key", []byte(`{}`), run)
	if e != nil || status != 202 {
		t.Fatalf("%d %v", status, e)
	}
	again, status, e := jobs.Start("a", a, "restart", "key", []byte(`{}`), run)
	if e != nil || status != 200 || again.ID != first.ID {
		t.Fatal("duplicate not reconciled")
	}
	if _, _, e = jobs.Start("a", a, "reset", "key", []byte(`{}`), run); e == nil {
		t.Fatal("mismatched key reused")
	}
	reopened, e := NewJobs(dir)
	if e != nil {
		t.Fatal(e)
	}
	got, _ := reopened.Get(first.ID)
	if got.State != "unknown" {
		t.Fatalf("crashed operation state=%s", got.State)
	}
	if _, found, e := reopened.Lookup("a", a, "restart", "key", []byte(`{}`)); !found || e != nil {
		t.Fatal("durable idempotency lost")
	}
	close(release)
	<-done
}
func TestJobFailureAndPanicReleaseAdmission(t *testing.T) {
	j, e := NewJobs(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	a := Actor{ID: "1"}
	v, _, _ := j.Start("a", a, "one", "one", nil, func(string) (any, error) { return nil, errors.New("expected") })
	waitJob(t, j, v.ID, "failed")
	v, _, e = j.Start("a", a, "two", "two", nil, func(string) (any, error) { panic("boom") })
	if e != nil {
		t.Fatal(e)
	}
	waitJob(t, j, v.ID, "unknown")
}
func waitJob(t *testing.T, j *Jobs, id, state string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		v, _ := j.Get(id)
		j.mu.Lock()
		busy := j.busy
		j.mu.Unlock()
		if v.State == state && busy == "" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	v, _ := j.Get(id)
	t.Fatalf("job state=%s want=%s", v.State, state)
}
