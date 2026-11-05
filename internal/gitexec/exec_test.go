package gitexec

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestRunnerVersionAndEnvironment(t *testing.T) {
	path, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	result, err := (Runner{Path: path}).Run(context.Background(), Command{Args: []string{"version"}})
	if err != nil || !strings.HasPrefix(string(result.Stdout), "git version ") {
		t.Fatalf("git version: result=%+v err=%v", result, err)
	}
	for _, entry := range []string{"LC_ALL=C", "LANGUAGE=", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0"} {
		if !contains(overrideEnv([]string{"LC_ALL=fr", "GIT_OPTIONAL_LOCKS=1"}, false), entry) {
			t.Fatalf("missing %q", entry)
		}
	}
	if contains(overrideEnv([]string{"GIT_OPTIONAL_LOCKS=1"}, true), "GIT_OPTIONAL_LOCKS=0") {
		t.Fatal("write command forced read-only locking behavior")
	}
	if contains(overrideEnv([]string{"GIT_OPTIONAL_LOCKS=0"}, true), "GIT_OPTIONAL_LOCKS=0") {
		t.Fatal("write command inherited read-only locking behavior")
	}
}

func contains(entries []string, want string) bool {
	for _, entry := range entries {
		if entry == want {
			return true
		}
	}
	return false
}
