package webcontrol

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestJobStorageFailureReleasesAdmissionAfterRecovery(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "jobs")
	j, err := NewJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	var executions atomic.Int32
	v, _, err := j.Start("a", Actor{ID: "123"}, "first", "first", nil, func(string) (any, error) {
		executions.Add(1)
		if err := os.Rename(dir, dir+".offline"); err != nil {
			return nil, err
		}
		return "done", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	waitJob(t, j, v.ID, "unknown")
	state, _ := j.Get(v.ID)
	if state.Phase != "storage_error" || state.Error == "" {
		t.Fatalf("storage failure not reported: %+v", state)
	}
	run := func(string) (any, error) { executions.Add(1); return nil, nil }
	again, status, err := j.Start("a", Actor{ID: "123"}, "first", "first", nil, run)
	if err != nil || status != 200 || again.ID != v.ID || again.State != "unknown" || executions.Load() != 1 {
		t.Fatalf("retry replayed failed checkpoint: status=%d job=%+v error=%v", status, again, err)
	}
	if _, status, err := j.Start("a", Actor{ID: "123"}, "second", "second", nil, run); err == nil || status != 503 {
		t.Fatalf("accepted work while storage unavailable: status=%d error=%v", status, err)
	}
	if err := os.Rename(dir+".offline", dir); err != nil {
		t.Fatal(err)
	}
	next, status, err := j.Start("a", Actor{ID: "123"}, "second", "second", nil, run)
	if err != nil {
		t.Fatalf("storage recovered but controls remain blocked: state=%s status=%d error=%v", state.State, status, err)
	}
	waitJob(t, j, next.ID, "succeeded")
	reopened, err := NewJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, found, err := reopened.Lookup("a", Actor{ID: "123"}, "first", "first", nil)
	if err != nil || !found || got.State != "unknown" || got.Phase != "storage_error" || executions.Load() != 2 {
		t.Fatalf("recovered checkpoint lost: %+v found=%t error=%v executions=%d", got, found, err, executions.Load())
	}
}
