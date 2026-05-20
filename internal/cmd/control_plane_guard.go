package cmd

import (
	"fmt"
	"os"
	"strings"
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
