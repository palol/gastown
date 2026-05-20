package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func remoteExecutorMode() bool {
	if strings.TrimSpace(os.Getenv("GT_REMOTE_EXECUTOR")) == "1" {
		return true
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GT_CONTROL_PLANE_MODE")))
	return mode == "remote-executor"
}

func requireLocalControlPlane(action string) error {
	if !remoteExecutorMode() {
		return nil
	}
	return fmt.Errorf("blocked in remote executor mode: %s (canonical town state stays on laptop)", action)
}

func commandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	var parts []string
	for c := cmd; c != nil; c = c.Parent() {
		name := strings.TrimSpace(c.Name())
		if name == "" || name == "gt" {
			continue
		}
		parts = append([]string{name}, parts...)
	}
	return strings.Join(parts, " ")
}

func isBlockedRemoteMutation(cmd *cobra.Command, args []string) bool {
	path := commandPath(cmd)
	switch path {
	case "sling",
		"close",
		"assign",
		"handoff",
		"unsling",
		"mail send",
		"convoy create",
		"convoy add",
		"convoy close",
		"convoy land",
		"convoy launch",
		"convoy stage",
		"convoy watch",
		"convoy unwatch",
		"scheduler run",
		"scheduler clear",
		"dog dispatch",
		"dog done",
		"dog clear":
		return true
	case "hook":
		// "gt hook" with no args is read-only status view.
		return len(args) > 0
	case "hook clear":
		return true
	default:
		return false
	}
}
