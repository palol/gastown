package capacity

import (
	"testing"
	"time"
)

func TestRunStateHappyPath(t *testing.T) {
	f := &SlingContextFields{}
	now := time.Date(2026, 5, 20, 22, 0, 0, 0, time.UTC)

	path := []string{
		RunStateQueued,
		RunStatePreflight,
		RunStateSyncing,
		RunStateDispatched,
		RunStateRunning,
		RunStateCollecting,
		RunStateSucceeded,
	}

	for _, state := range path {
		if err := AdvanceRunState(f, state, now); err != nil {
			t.Fatalf("advance to %s failed: %v", state, err)
		}
		now = now.Add(time.Minute)
	}
}

func TestRunStateRejectInvalidTransition(t *testing.T) {
	f := &SlingContextFields{RunState: RunStateQueued}
	if err := AdvanceRunState(f, RunStateRunning, time.Now()); err == nil {
		t.Fatal("expected transition error")
	}
}
