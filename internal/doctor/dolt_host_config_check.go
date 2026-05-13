package doctor

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// DoltHostConfigCheck detects remote Dolt host settings that are only present
// in ephemeral env vars instead of the daemon config that spawns agents.
type DoltHostConfigCheck struct {
	BaseCheck
}

type daemonDoltHostConfig struct {
	Host   string
	Exists bool
}

// NewDoltHostConfigCheck creates a new Dolt host config check.
func NewDoltHostConfigCheck() *DoltHostConfigCheck {
	return &DoltHostConfigCheck{
		BaseCheck: BaseCheck{
			CheckName:        "dolt-host-config",
			CheckDescription: "Detect Dolt host settings that are not persisted in daemon.json",
			CheckCategory:    CategoryConfig,
		},
	}
}

// Run checks that remote Dolt host configuration is discoverable and propagated.
func (c *DoltHostConfigCheck) Run(ctx *CheckContext) *CheckResult {
	gtHost := strings.TrimSpace(os.Getenv("GT_DOLT_HOST"))
	beadsHost := strings.TrimSpace(os.Getenv("BEADS_DOLT_SERVER_HOST"))
	daemonHost := readDaemonDoltHost(ctx.TownRoot)

	var details []string
	var desiredHost string

	if isRemoteDoltHost(gtHost) {
		desiredHost = gtHost
		if !daemonHost.Exists || !sameHost(daemonHost.Host, gtHost) {
			details = append(details, fmt.Sprintf("GT_DOLT_HOST=%q is set, but mayor/daemon.json does not persist patrols.dolt_server.host", gtHost))
		}
	}

	if isRemoteDoltHost(daemonHost.Host) && desiredHost == "" {
		desiredHost = daemonHost.Host
	}

	if desiredHost != "" && beadsHost != "" && !sameHost(beadsHost, desiredHost) {
		details = append(details, fmt.Sprintf("BEADS_DOLT_SERVER_HOST=%q does not match remote Dolt host %q", beadsHost, desiredHost))
	}

	if daemonHost.Exists && isWildcardDoltHost(daemonHost.Host) {
		details = append(details, fmt.Sprintf("mayor/daemon.json patrols.dolt_server.host is %q, which is a bind address, not a client connection host", daemonHost.Host))
	}

	if len(details) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "Dolt host configuration is consistent",
		}
	}

	snippetHost := desiredHost
	if snippetHost == "" {
		snippetHost = "<remote-host>"
	}
	details = append(details, "", "Persist the remote host in mayor/daemon.json:", daemonDoltHostSnippet(snippetHost))

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusWarning,
		Message: "Remote Dolt host is not consistently configured",
		Details: details,
		FixHint: "Add patrols.dolt_server.host to mayor/daemon.json and restart Gas Town",
	}
}

func readDaemonDoltHost(townRoot string) daemonDoltHostConfig {
	data, err := os.ReadFile(filepath.Join(townRoot, "mayor", "daemon.json")) //nolint:gosec // G304: path is within town root
	if err != nil {
		return daemonDoltHostConfig{}
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return daemonDoltHostConfig{}
	}

	if host, ok := doltHostFromRaw(raw["dolt_server"]); ok {
		return daemonDoltHostConfig{Host: host, Exists: true}
	}

	var patrols map[string]json.RawMessage
	if err := json.Unmarshal(raw["patrols"], &patrols); err != nil {
		return daemonDoltHostConfig{}
	}
	if host, ok := doltHostFromRaw(patrols["dolt_server"]); ok {
		return daemonDoltHostConfig{Host: host, Exists: true}
	}

	return daemonDoltHostConfig{}
}

func doltHostFromRaw(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var cfg struct {
		Host string `json:"host"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", false
	}
	return strings.TrimSpace(cfg.Host), true
}

func isRemoteDoltHost(host string) bool {
	host = normalizeDoltHost(host)
	return host != "" && !isLocalDoltHost(host) && !isWildcardDoltHost(host)
}

func isLocalDoltHost(host string) bool {
	host = normalizeDoltHost(host)
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isWildcardDoltHost(host string) bool {
	host = normalizeDoltHost(host)
	return host == "0.0.0.0" || host == "::"
}

func sameHost(a, b string) bool {
	return normalizeDoltHost(a) == normalizeDoltHost(b)
}

func normalizeDoltHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return host
}

func daemonDoltHostSnippet(host string) string {
	return fmt.Sprintf(`"patrols": {
  "dolt_server": {
    "enabled": true,
    "external": true,
    "host": %q,
    "port": 3307
  }
}`, host)
}
