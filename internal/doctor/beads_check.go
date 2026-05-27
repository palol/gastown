package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/beads"
)

// PrefixConflictCheck detects duplicate prefixes across rigs in routes.jsonl.
// Duplicate prefixes break prefix-based routing.
type PrefixConflictCheck struct {
	BaseCheck
}

// NewPrefixConflictCheck creates a new prefix conflict check.
func NewPrefixConflictCheck() *PrefixConflictCheck {
	return &PrefixConflictCheck{
		BaseCheck: BaseCheck{
			CheckName:        "prefix-conflict",
			CheckDescription: "Check for duplicate beads prefixes across rigs",
			CheckCategory:    CategoryConfig,
		},
	}
}

// Run checks for duplicate prefixes in routes.jsonl.
func (c *PrefixConflictCheck) Run(ctx *CheckContext) *CheckResult {
	beadsDir := filepath.Join(ctx.TownRoot, ".beads")

	// Check if routes.jsonl exists
	routesPath := filepath.Join(beadsDir, beads.RoutesFileName)
	if _, err := os.Stat(routesPath); os.IsNotExist(err) {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No routes.jsonl file (prefix routing not configured)",
		}
	}

	// Find conflicts
	conflicts, err := beads.FindConflictingPrefixes(beadsDir)
	if err != nil {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not check routes.jsonl: %v", err),
		}
	}

	if len(conflicts) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No prefix conflicts found",
		}
	}

	// Build details
	var details []string
	for prefix, paths := range conflicts {
		details = append(details, fmt.Sprintf("Prefix %q used by: %s", prefix, strings.Join(paths, ", ")))
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusError,
		Message: fmt.Sprintf("%d prefix conflict(s) found in routes.jsonl", len(conflicts)),
		Details: details,
		FixHint: "Use 'bd rename-prefix <new-prefix>' in one of the conflicting rigs to resolve",
	}
}

// PrefixMismatchCheck detects when rigs.json has a different prefix than what
// routes.jsonl actually uses for a rig. This can happen when:
// - deriveBeadsPrefix() generates a different prefix than what's in the beads DB
// - Someone manually edited rigs.json with the wrong prefix
// - The beads were initialized before auto-derive existed with a different prefix
type PrefixMismatchCheck struct {
	FixableCheck
}

// NewPrefixMismatchCheck creates a new prefix mismatch check.
func NewPrefixMismatchCheck() *PrefixMismatchCheck {
	return &PrefixMismatchCheck{
		FixableCheck: FixableCheck{
			BaseCheck: BaseCheck{
				CheckName:        "prefix-mismatch",
				CheckDescription: "Check for prefix mismatches between rigs.json and routes.jsonl",
				CheckCategory:    CategoryConfig,
			},
		},
	}
}

// Run checks for prefix mismatches between rigs.json and routes.jsonl.
func (c *PrefixMismatchCheck) Run(ctx *CheckContext) *CheckResult {
	beadsDir := filepath.Join(ctx.TownRoot, ".beads")

	// Load routes.jsonl
	routes, err := beads.LoadRoutes(beadsDir)
	if err != nil {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not load routes.jsonl: %v", err),
		}
	}
	if len(routes) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No routes configured (nothing to check)",
		}
	}

	// Load rigs.json
	rigsPath := filepath.Join(ctx.TownRoot, "mayor", "rigs.json")
	rigsConfig, err := loadRigsConfig(rigsPath)
	if err != nil {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No rigs.json found (nothing to check)",
		}
	}

	// Build map of route path -> prefix from routes.jsonl
	routePrefixByPath := make(map[string]string)
	for _, r := range routes {
		// Normalize: strip trailing hyphen from prefix for comparison
		prefix := strings.TrimSuffix(r.Prefix, "-")
		routePrefixByPath[r.Path] = prefix
	}

	// Check each rig in rigs.json against routes.jsonl
	var mismatches []string
	mismatchData := make(map[string][2]string) // rigName -> [rigsJsonPrefix, routesPrefix]

	for rigName, rigEntry := range rigsConfig.Rigs {
		// Skip rigs without beads config
		if rigEntry.BeadsConfig == nil || rigEntry.BeadsConfig.Prefix == "" {
			continue
		}

		rigsJsonPrefix := rigEntry.BeadsConfig.Prefix
		expectedPath := determineRigBeadsPath(ctx.TownRoot, rigName)

		// Find the route for this rig
		routePrefix, hasRoute := routePrefixByPath[expectedPath]
		if !hasRoute {
			// No route for this rig - routes-config check handles this
			continue
		}

		// Compare prefixes (both should be without trailing hyphen)
		if rigsJsonPrefix != routePrefix {
			mismatches = append(mismatches, rigName)
			mismatchData[rigName] = [2]string{rigsJsonPrefix, routePrefix}
		}
	}

	if len(mismatches) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No prefix mismatches found",
		}
	}

	// Build details
	var details []string
	for _, rigName := range mismatches {
		data := mismatchData[rigName]
		details = append(details, fmt.Sprintf("Rig '%s': rigs.json says '%s', routes.jsonl uses '%s'",
			rigName, data[0], data[1]))
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusWarning,
		Message: fmt.Sprintf("%d prefix mismatch(es) between rigs.json and routes.jsonl", len(mismatches)),
		Details: details,
		FixHint: "Run 'gt doctor --fix' to update rigs.json with correct prefixes",
	}
}

// Fix updates rigs.json to match the prefixes in routes.jsonl.
func (c *PrefixMismatchCheck) Fix(ctx *CheckContext) error {
	beadsDir := filepath.Join(ctx.TownRoot, ".beads")

	// Load routes.jsonl
	routes, err := beads.LoadRoutes(beadsDir)
	if err != nil || len(routes) == 0 {
		return nil // Nothing to fix
	}

	// Load rigs.json
	rigsPath := filepath.Join(ctx.TownRoot, "mayor", "rigs.json")
	rigsConfig, err := loadRigsConfig(rigsPath)
	if err != nil {
		return nil // Nothing to fix
	}

	// Build map of route path -> prefix from routes.jsonl
	routePrefixByPath := make(map[string]string)
	for _, r := range routes {
		prefix := strings.TrimSuffix(r.Prefix, "-")
		routePrefixByPath[r.Path] = prefix
	}

	// Update each rig's prefix to match routes.jsonl
	modified := false
	for rigName, rigEntry := range rigsConfig.Rigs {
		expectedPath := determineRigBeadsPath(ctx.TownRoot, rigName)
		routePrefix, hasRoute := routePrefixByPath[expectedPath]
		if !hasRoute {
			continue
		}

		// Ensure BeadsConfig exists
		if rigEntry.BeadsConfig == nil {
			rigEntry.BeadsConfig = &rigsConfigBeadsConfig{}
		}

		if rigEntry.BeadsConfig.Prefix != routePrefix {
			rigEntry.BeadsConfig.Prefix = routePrefix
			rigsConfig.Rigs[rigName] = rigEntry
			modified = true
		}
	}

	if modified {
		return saveRigsConfig(rigsPath, rigsConfig)
	}

	return nil
}

// rigsConfigEntry is a local type for loading rigs.json without importing config package
// to avoid circular dependencies and keep the check self-contained.
type rigsConfigEntry struct {
	GitURL      string                 `json:"git_url"`
	LocalRepo   string                 `json:"local_repo,omitempty"`
	AddedAt     string                 `json:"added_at"` // Keep as string to preserve format
	BeadsConfig *rigsConfigBeadsConfig `json:"beads,omitempty"`
}

type rigsConfigBeadsConfig struct {
	Repo   string `json:"repo"`
	Prefix string `json:"prefix"`
}

type rigsConfigFile struct {
	Version int                        `json:"version"`
	Rigs    map[string]rigsConfigEntry `json:"rigs"`
}

func loadRigsConfig(path string) (*rigsConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg rigsConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func saveRigsConfig(path string, cfg *rigsConfigFile) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// dbPrefixGetter abstracts querying the database for issue_prefix.
// Allows mocking in tests without shelling out to bd.
type dbPrefixGetter interface {
	GetDBPrefix(rigPath string) (string, error)
}

// realDBPrefixGetter shells out to bd to query the database.
type realDBPrefixGetter struct{}

func (r *realDBPrefixGetter) GetDBPrefix(rigPath string) (string, error) {
	cmd := exec.Command("bd", "config", "get", "issue_prefix")
	cmd.Dir = rigPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return normalizeDBPrefix(string(output)), nil
}

// normalizeDBPrefix interprets the stdout of "bd config get issue_prefix".
// When the key is unset, bd prints "<key> (not set)" to stdout and exits 0;
// normalize that (and an empty result) to "" so callers can treat it uniformly
// as "no prefix set" instead of comparing against a human-readable sentinel.
func normalizeDBPrefix(output string) string {
	prefix := strings.TrimSpace(output)
	if prefix == "" || strings.HasSuffix(prefix, "(not set)") {
		return ""
	}
	return prefix
}

// DatabasePrefixCheck detects when a rig's database has a different issue_prefix
// than what routes.jsonl specifies. This can happen when:
// - The database was initialized with a different prefix
// - Manual database edits changed the prefix
// - A bug in prefix derivation caused a mismatch
//
// Unlike PrefixMismatchCheck (rigs.json ↔ routes.jsonl), this check verifies
// the actual database configuration matches the routing table.
//
// Rigs that redirect to a shared database (e.g. the town root's .beads) are
// skipped. Their database prefix is owned by the route that provides the
// canonical database, not by the redirecting rig. Attempting to "fix" these
// would overwrite the shared database's prefix with the rig's prefix.
type DatabasePrefixCheck struct {
	FixableCheck
	mismatches   []databasePrefixMismatch
	prefixGetter dbPrefixGetter
}

type databasePrefixMismatch struct {
	rigPath      string
	routesPrefix string // From routes.jsonl (without trailing hyphen)
	dbPrefix     string // From database config ("" when unset)
	unset        bool   // true when the database has no issue_prefix set
}

// NewDatabasePrefixCheck creates a new database prefix check.
func NewDatabasePrefixCheck() *DatabasePrefixCheck {
	return &DatabasePrefixCheck{
		FixableCheck: FixableCheck{
			BaseCheck: BaseCheck{
				CheckName:        "database-prefix",
				CheckDescription: "Check rig database issue_prefix matches routes.jsonl",
				CheckCategory:    CategoryConfig,
			},
		},
	}
}

// Run checks if each rig's database issue_prefix matches routes.jsonl.
func (c *DatabasePrefixCheck) Run(ctx *CheckContext) *CheckResult {
	c.mismatches = nil // Reset

	beadsDir := filepath.Join(ctx.TownRoot, ".beads")

	// Load routes.jsonl
	routes, err := beads.LoadRoutes(beadsDir)
	if err != nil {
		return &CheckResult{
			Name:     c.Name(),
			Status:   StatusOK,
			Message:  "No routes.jsonl found (nothing to check)",
			Category: c.Category(),
		}
	}
	if len(routes) == 0 {
		return &CheckResult{
			Name:     c.Name(),
			Status:   StatusOK,
			Message:  "No routes configured (nothing to check)",
			Category: c.Category(),
		}
	}

	// Check if bd command is available (skip when using injected mock)
	if c.prefixGetter == nil {
		if _, err := exec.LookPath("bd"); err != nil {
			return &CheckResult{
				Name:     c.Name(),
				Status:   StatusOK,
				Message:  "beads not installed (skipped)",
				Category: c.Category(),
			}
		}
	}

	getter := c.prefixGetter
	if getter == nil {
		getter = &realDBPrefixGetter{}
	}

	// Resolve the town root's canonical beads directory so we can detect
	// rigs that redirect to the shared town database.
	townBeadsDir, _ := filepath.Abs(beads.ResolveBeadsDir(ctx.TownRoot))

	var problems []string

	for _, route := range routes {
		// Skip town root route
		if route.Path == "." || route.Path == "" {
			continue
		}

		rigPath := filepath.Join(ctx.TownRoot, route.Path)
		rigBeadsDir := beads.ResolveBeadsDir(rigPath)

		// Check if beads directory exists
		if _, err := os.Stat(rigBeadsDir); os.IsNotExist(err) {
			continue
		}

		// Skip rigs whose beads redirect resolves to the town root database.
		// These rigs share the town DB; the prefix is owned by the town root
		// route, not by this rig. "Fixing" them would overwrite the shared
		// database's issue_prefix with the rig's route prefix.
		absRigBeadsDir, _ := filepath.Abs(rigBeadsDir)
		if absRigBeadsDir == townBeadsDir {
			continue
		}

		dbPrefix, err := getter.GetDBPrefix(rigPath)
		if err != nil {
			continue
		}

		routesPrefix := strings.TrimSuffix(route.Prefix, "-")

		if dbPrefix != routesPrefix {
			unset := dbPrefix == ""
			if unset {
				problems = append(problems, fmt.Sprintf("Route '%s': routes.jsonl says '%s', database has no issue_prefix set",
					route.Path, routesPrefix))
			} else {
				problems = append(problems, fmt.Sprintf("Route '%s': routes.jsonl says '%s', database has '%s'",
					route.Path, routesPrefix, dbPrefix))
			}
			c.mismatches = append(c.mismatches, databasePrefixMismatch{
				rigPath:      route.Path,
				routesPrefix: routesPrefix,
				dbPrefix:     dbPrefix,
				unset:        unset,
			})
		}
	}

	if len(c.mismatches) == 0 {
		return &CheckResult{
			Name:     c.Name(),
			Status:   StatusOK,
			Message:  "All database prefixes match routes.jsonl",
			Category: c.Category(),
		}
	}

	return &CheckResult{
		Name:     c.Name(),
		Status:   StatusWarning,
		Message:  fmt.Sprintf("%d database prefix mismatch(es) with routes.jsonl", len(c.mismatches)),
		Details:  problems,
		FixHint:  "Changing a database's issue_prefix is manual: for an unset prefix run 'bd bootstrap' (or 'bd init --prefix <p>-') in the rig; for a wrong-but-set prefix run 'bd rename-prefix <p>-'. 'gt doctor --fix' prints per-rig guidance but does not change the prefix.",
		Category: c.Category(),
	}
}

// Fix is guidance-only: it prints the correct manual command for each mismatch
// and returns nil without mutating any database.
//
// It deliberately does NOT run "bd config set issue_prefix": bd rejects that
// command (issue_prefix is settable only via init/bootstrap/rename-prefix), so
// the previous implementation could never succeed. The correct command also
// depends on the situation — an unset prefix needs 'bd bootstrap'/'bd init',
// while a wrong-but-set prefix needs 'bd rename-prefix', which rewrites every
// issue ID. Both are sensitive enough to leave to a human. The mismatch is not
// auto-resolved and will continue to be reported until fixed manually.
func (c *DatabasePrefixCheck) Fix(ctx *CheckContext) error {
	if len(c.mismatches) == 0 {
		result := c.Run(ctx)
		if result.Status == StatusOK {
			return nil
		}
	}

	for _, m := range c.mismatches {
		if m.unset {
			fmt.Fprintf(os.Stderr, "database-prefix: %s has no issue_prefix set; routes.jsonl expects %q.\n"+
				"  Initialize it manually, e.g. run 'bd bootstrap' (or 'bd init --prefix %s-') in %s.\n",
				m.rigPath, m.routesPrefix, m.routesPrefix, m.rigPath)
		} else {
			fmt.Fprintf(os.Stderr, "database-prefix: %s has issue_prefix %q but routes.jsonl expects %q.\n"+
				"  Rename it manually: run 'bd rename-prefix %s-' in %s.\n",
				m.rigPath, m.dbPrefix, m.routesPrefix, m.routesPrefix, m.rigPath)
		}
	}

	return nil
}
