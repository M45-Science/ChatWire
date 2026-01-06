package support

import "testing"

func TestRunHandlesStopsAtFirstMatch(t *testing.T) {
	calls := 0
	handles := []funcList{
		{function: func(*handleData) bool {
			calls++
			return true
		}},
		{function: func(*handleData) bool {
			calls++
			return true
		}},
	}

	runHandles(handles, &handleData{})
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}
