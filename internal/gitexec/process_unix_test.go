//go:build linux || darwin

package gitexec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCanceledHookChildCannotFinish(t *testing.T) {
	path, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	setup := exec.Command(path, "init", dir)
	if output, err := setup.CombinedOutput(); err != nil {
		t.Fatalf("init: %v: %s", err, output)
	}
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	started := filepath.Join(dir, "started")
	finished := filepath.Join(dir, "finished")
	contents := "#!/bin/sh\n(printf ready > '" + started + "'; sleep 2; printf finished > '" + finished + "') &\nwait\n"
	if err := os.WriteFile(hook, []byte(contents), 0755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := (Runner{Path: path}).Run(ctx, Command{Dir: dir, Args: []string{"-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "test"}, Write: true})
		done <- err
	}()
	deadline := time.After(5 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("hook never started")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "git") {
			t.Fatalf("cancel result: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("git did not stop")
	}
	select {
	case <-time.After(2300 * time.Millisecond):
	}
	if _, err := os.Stat(finished); err == nil {
		t.Fatal("hook child survived cancellation")
	}
}
