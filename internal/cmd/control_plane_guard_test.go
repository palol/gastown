package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

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

func TestIsBlockedRemoteMutation(t *testing.T) {
	root := &cobra.Command{Use: "gt"}
	mail := &cobra.Command{Use: "mail"}
	send := &cobra.Command{Use: "send"}
	hook := &cobra.Command{Use: "hook"}
	mail.AddCommand(send)
	root.AddCommand(mail, hook)

	if !isBlockedRemoteMutation(send, nil) {
		t.Fatal("mail send should be blocked")
	}
	if isBlockedRemoteMutation(hook, nil) {
		t.Fatal("hook with no args should not be blocked")
	}
	if !isBlockedRemoteMutation(hook, []string{"gt-123"}) {
		t.Fatal("hook with args should be blocked")
	}
}
