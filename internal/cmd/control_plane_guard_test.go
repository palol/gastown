package cmd

import "testing"

func TestRequireLocalControlPlane(t *testing.T) {
	t.Setenv("GT_REMOTE_EXECUTOR", "")
	t.Setenv("GT_CONTROL_PLANE_MODE", "")
	if err := requireLocalControlPlane("sling"); err != nil {
		t.Fatalf("unexpected error in local mode: %v", err)
	}

	t.Setenv("GT_REMOTE_EXECUTOR", "1")
	if err := requireLocalControlPlane("sling"); err == nil {
		t.Fatal("expected block in remote executor mode")
	}

	t.Setenv("GT_REMOTE_EXECUTOR", "")
	t.Setenv("GT_CONTROL_PLANE_MODE", "remote-executor")
	if err := requireLocalControlPlane("mail send"); err == nil {
		t.Fatal("expected block in remote control plane mode")
	}
}
