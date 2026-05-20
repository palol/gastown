package capacity

import (
	"fmt"
	"time"
)

const (
	RunStateQueued     = "queued"
	RunStatePreflight  = "preflight"
	RunStateSyncing    = "syncing"
	RunStateDispatched = "dispatched"
	RunStateRunning    = "running"
	RunStateCollecting = "collecting"
	RunStateSucceeded  = "succeeded"
	RunStateFailed     = "failed"
	RunStateCancelled  = "cancelled"
	RunStateAbandoned  = "abandoned"
)

var runStateTransitions = map[string]map[string]bool{
	"":                 {RunStateQueued: true},
	RunStateQueued:     {RunStatePreflight: true, RunStateCancelled: true},
	RunStatePreflight:  {RunStateSyncing: true, RunStateFailed: true, RunStateCancelled: true},
	RunStateSyncing:    {RunStateDispatched: true, RunStateFailed: true, RunStateCancelled: true},
	RunStateDispatched: {RunStateRunning: true, RunStateFailed: true, RunStateCancelled: true},
	RunStateRunning:    {RunStateCollecting: true, RunStateFailed: true, RunStateAbandoned: true, RunStateCancelled: true},
	RunStateCollecting: {RunStateSucceeded: true, RunStateFailed: true, RunStateAbandoned: true, RunStateCancelled: true},
	RunStateSucceeded:  {},
	RunStateFailed:     {},
	RunStateCancelled:  {},
	RunStateAbandoned:  {},
}

func IsValidRunStateTransition(from, to string) bool {
	next, ok := runStateTransitions[from]
	if !ok {
		return false
	}
	return next[to]
}

func AdvanceRunState(fields *SlingContextFields, to string, at time.Time) error {
	if fields == nil {
		return fmt.Errorf("nil sling context")
	}
	if !IsValidRunStateTransition(fields.RunState, to) {
		return fmt.Errorf("invalid run state transition: %s -> %s", fields.RunState, to)
	}
	fields.RunState = to
	fields.RunStateUpdatedAt = at.UTC().Format(time.RFC3339)
	switch to {
	case RunStateRunning:
		fields.LastHeartbeatAt = at.UTC().Format(time.RFC3339)
	case RunStateCollecting:
		fields.CollectedAt = ""
	case RunStateSucceeded:
		fields.CollectedAt = at.UTC().Format(time.RFC3339)
	}
	return nil
}

func TouchRunHeartbeat(fields *SlingContextFields, at time.Time) {
	if fields == nil {
		return
	}
	fields.LastHeartbeatAt = at.UTC().Format(time.RFC3339)
}
