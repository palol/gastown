package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steveyegge/gastown/internal/scheduler/capacity"
)

func TestDetermineRunReaperAction_StaleHeartbeat(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	fields := &capacity.SlingContextFields{
		RunState:        capacity.RunStateRunning,
		LastHeartbeatAt: now.Add(-runningHeartbeatTimeout - time.Minute).Format(time.RFC3339),
	}
	got := determineRunReaperAction(fields, now)
	if got.CloseReason != "stale-heartbeat" {
		t.Fatalf("close reason = %q, want stale-heartbeat", got.CloseReason)
	}
	if got.NextState != capacity.RunStateAbandoned {
		t.Fatalf("next state = %q, want %q", got.NextState, capacity.RunStateAbandoned)
	}
}

func TestDetermineRunReaperAction_CollectTimeout(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	fields := &capacity.SlingContextFields{
		RunState:          capacity.RunStateCollecting,
		RunStateUpdatedAt: now.Add(-collectTimeout - time.Minute).Format(time.RFC3339),
	}
	got := determineRunReaperAction(fields, now)
	if got.CloseReason != "collect-timeout" {
		t.Fatalf("close reason = %q, want collect-timeout", got.CloseReason)
	}
	if got.NextState != capacity.RunStateAbandoned {
		t.Fatalf("next state = %q, want %q", got.NextState, capacity.RunStateAbandoned)
	}
}

func TestCollectAndAuditBurstArtifacts(t *testing.T) {
	townRoot := t.TempDir()
	inbox := filepath.Join(townRoot, "runs")
	t.Setenv("GT_BIGFOOT_RUN_INBOX", inbox)
	runID := "run-abc"
	runDir := filepath.Join(inbox, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "manifest.json"), []byte(`{"files":["summary.md"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "summary.md"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "patch.diff"), []byte("diff --git"), 0o644); err != nil {
		t.Fatal(err)
	}

	fields := &capacity.SlingContextFields{DispatchRunID: runID}
	ready, err := collectAndAuditBurstArtifacts(townRoot, fields)
	if err != nil {
		t.Fatalf("collectAndAuditBurstArtifacts err: %v", err)
	}
	if !ready {
		t.Fatal("expected ready=true")
	}
	if fields.ManifestSHA256 == "" || fields.SummarySHA256 == "" || fields.PatchSHA256 == "" {
		t.Fatal("expected sha256 hashes to be filled")
	}
}
