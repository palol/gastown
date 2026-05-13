package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoltHostConfigCheck_NoRemoteHost(t *testing.T) {
	t.Setenv("GT_DOLT_HOST", "")
	t.Setenv("BEADS_DOLT_SERVER_HOST", "")

	check := NewDoltHostConfigCheck()
	result := check.Run(&CheckContext{TownRoot: t.TempDir()})

	if result.Status != StatusOK {
		t.Fatalf("expected OK, got %s: %s", result.Status, result.Message)
	}
}

func TestDoltHostConfigCheck_WarnsWhenEnvHostNotPersisted(t *testing.T) {
	t.Setenv("GT_DOLT_HOST", "100.111.197.110")
	t.Setenv("BEADS_DOLT_SERVER_HOST", "100.111.197.110")

	check := NewDoltHostConfigCheck()
	result := check.Run(&CheckContext{TownRoot: t.TempDir()})

	if result.Status != StatusWarning {
		t.Fatalf("expected Warning, got %s: %s", result.Status, result.Message)
	}
	if !strings.Contains(strings.Join(result.Details, "\n"), "100.111.197.110") {
		t.Fatalf("expected details to include remote host, got %v", result.Details)
	}
}

func TestDoltHostConfigCheck_WarnsOnWildcardDaemonHost(t *testing.T) {
	t.Setenv("GT_DOLT_HOST", "100.111.197.110")
	t.Setenv("BEADS_DOLT_SERVER_HOST", "100.111.197.110")
	townRoot := t.TempDir()
	writeDaemonJSON(t, townRoot, `{
  "patrols": {
    "dolt_server": {"enabled": true, "external": true, "host": "0.0.0.0", "port": 3307}
  }
}`)

	check := NewDoltHostConfigCheck()
	result := check.Run(&CheckContext{TownRoot: townRoot})

	if result.Status != StatusWarning {
		t.Fatalf("expected Warning, got %s: %s", result.Status, result.Message)
	}
	if !strings.Contains(strings.Join(result.Details, "\n"), "bind address") {
		t.Fatalf("expected bind-address warning, got %v", result.Details)
	}
}

func TestDoltHostConfigCheck_OKWhenEnvHostPersisted(t *testing.T) {
	t.Setenv("GT_DOLT_HOST", "100.111.197.110")
	t.Setenv("BEADS_DOLT_SERVER_HOST", "100.111.197.110")
	townRoot := t.TempDir()
	writeDaemonJSON(t, townRoot, `{
  "patrols": {
    "dolt_server": {"enabled": true, "external": true, "host": "100.111.197.110", "port": 3307}
  }
}`)

	check := NewDoltHostConfigCheck()
	result := check.Run(&CheckContext{TownRoot: townRoot})

	if result.Status != StatusOK {
		t.Fatalf("expected OK, got %s: %s details=%v", result.Status, result.Message, result.Details)
	}
}

func TestDoltHostConfigCheck_WarnsOnBeadsEnvMismatch(t *testing.T) {
	t.Setenv("GT_DOLT_HOST", "100.111.197.110")
	t.Setenv("BEADS_DOLT_SERVER_HOST", "127.0.0.1")
	townRoot := t.TempDir()
	writeDaemonJSON(t, townRoot, `{
  "patrols": {
    "dolt_server": {"enabled": true, "external": true, "host": "100.111.197.110", "port": 3307}
  }
}`)

	check := NewDoltHostConfigCheck()
	result := check.Run(&CheckContext{TownRoot: townRoot})

	if result.Status != StatusWarning {
		t.Fatalf("expected Warning, got %s: %s", result.Status, result.Message)
	}
	if !strings.Contains(strings.Join(result.Details, "\n"), "BEADS_DOLT_SERVER_HOST") {
		t.Fatalf("expected BEADS_DOLT_SERVER_HOST mismatch, got %v", result.Details)
	}
}

func writeDaemonJSON(t *testing.T, townRoot, content string) {
	t.Helper()
	mayorDir := filepath.Join(townRoot, "mayor")
	if err := os.MkdirAll(mayorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mayorDir, "daemon.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
