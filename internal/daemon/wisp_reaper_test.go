package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWispReaperInterval(t *testing.T) {
	// Default (now 1h after Dog-driven refactor)
	if got := wispReaperInterval(nil); got != defaultWispReaperInterval {
		t.Errorf("expected default %v, got %v", defaultWispReaperInterval, got)
	}

	// Custom
	config := &DaemonPatrolConfig{
		Patrols: &PatrolsConfig{
			WispReaper: &WispReaperConfig{
				Enabled:     true,
				IntervalStr: "2h",
			},
		},
	}
	if got := wispReaperInterval(config); got != 2*time.Hour {
		t.Errorf("expected 2h, got %v", got)
	}

	// Invalid falls back to default
	config.Patrols.WispReaper.IntervalStr = "nope"
	if got := wispReaperInterval(config); got != defaultWispReaperInterval {
		t.Errorf("expected default for invalid, got %v", got)
	}
}

func TestWispReaperMaxAge(t *testing.T) {
	if got := wispReaperMaxAge(nil); got != defaultWispMaxAge {
		t.Errorf("expected default %v, got %v", defaultWispMaxAge, got)
	}

	config := &DaemonPatrolConfig{
		Patrols: &PatrolsConfig{
			WispReaper: &WispReaperConfig{
				Enabled:   true,
				MaxAgeStr: "48h",
			},
		},
	}
	if got := wispReaperMaxAge(config); got != 48*time.Hour {
		t.Errorf("expected 48h, got %v", got)
	}
}

func TestWispDeleteAge(t *testing.T) {
	if got := wispDeleteAge(nil); got != defaultWispDeleteAge {
		t.Errorf("expected default %v, got %v", defaultWispDeleteAge, got)
	}

	config := &DaemonPatrolConfig{
		Patrols: &PatrolsConfig{
			WispReaper: &WispReaperConfig{
				Enabled:      true,
				DeleteAgeStr: "336h",
			},
		},
	}
	if got := wispDeleteAge(config); got != 14*24*time.Hour {
		t.Errorf("expected 336h, got %v", got)
	}
}

func TestDefaultReaperIntervalIsOneHour(t *testing.T) {
	// Verify the default changed from 30m to 1h per issue gt-caf7.
	if defaultWispReaperInterval != 1*time.Hour {
		t.Errorf("expected default interval 1h, got %v", defaultWispReaperInterval)
	}
}

func TestDoltServerPortFallsBackToTownConfig(t *testing.T) {
	townRoot := t.TempDir()
	doltDataDir := filepath.Join(townRoot, ".dolt-data")
	if err := os.MkdirAll(doltDataDir, 0755); err != nil {
		t.Fatalf("create .dolt-data: %v", err)
	}
	configYAML := []byte("listener:\n  host: 127.0.0.1\n  port: 21307\n")
	if err := os.WriteFile(filepath.Join(doltDataDir, "config.yaml"), configYAML, 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	d := &Daemon{config: &Config{TownRoot: townRoot}}
	if got := d.doltServerPort(); got != 21307 {
		t.Errorf("doltServerPort() = %d, want 21307", got)
	}
}

func TestDoltServerPortFallsBackToEnv(t *testing.T) {
	t.Setenv("GT_DOLT_PORT", "21307")
	d := &Daemon{}
	if got := d.doltServerPort(); got != 21307 {
		t.Errorf("doltServerPort() = %d, want 21307", got)
	}
}

func TestDoltServerPortFallsBackToBeadsPortFile(t *testing.T) {
	townRoot := t.TempDir()
	beadsDir := filepath.Join(townRoot, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("create .beads: %v", err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "dolt-server.port"), []byte("21307\n"), 0644); err != nil {
		t.Fatalf("write dolt-server.port: %v", err)
	}

	d := &Daemon{config: &Config{TownRoot: townRoot}}
	if got := d.doltServerPort(); got != 21307 {
		t.Errorf("doltServerPort() = %d, want 21307", got)
	}
}
