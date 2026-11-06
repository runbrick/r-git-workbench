package gitexec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestProgressOutputSplitsCRLinesAndFlushesRemainder(t *testing.T) {
	var captured limitedOutput
	captured.limit = 1024
	var lines []string
	progress := progressOutput{output: &captured, notify: func(line string) { lines = append(lines, line) }}
	if _, err := progress.Write([]byte("Counting objects: 10%\rCounting objects: 100%\nWriting objects: 50%")); err != nil {
		t.Fatal(err)
	}
	progress.flush()
	if got := strings.Join(lines, "|"); got != "Counting objects: 10%|Counting objects: 100%|Writing objects: 50%" {
		t.Fatalf("progress lines = %q", got)
	}
	if !bytes.Equal(captured.buffer.Bytes(), []byte("Counting objects: 10%\rCounting objects: 100%\nWriting objects: 50%")) {
		t.Fatalf("stderr capture changed: %q", captured.buffer.Bytes())
	}
}

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

func TestRunnerOutputLimit(t *testing.T) {
	path, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	runner := Runner{Path: path}
	result, err := runner.Run(context.Background(), Command{Args: []string{"version"}, OutputLimit: 4})
	if err == nil || !strings.Contains(err.Error(), "capture limit") || len(result.Stdout) != 4 {
		t.Fatalf("limited output: %+v, %v", result, err)
	}
	result, err = runner.Run(context.Background(), Command{Args: []string{"version"}})
	if err != nil || !strings.HasPrefix(string(result.Stdout), "git version ") {
		t.Fatalf("subsequent command: %+v, %v", result, err)
	}
}

func TestRunnerDoesNotStartCanceledCommand(t *testing.T) {
	path, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := (Runner{Path: path}).Run(ctx, Command{Args: []string{"version"}})
	if !errors.Is(err, context.Canceled) || result.ExitCode != -1 || len(result.Stdout) != 0 {
		t.Fatalf("canceled command: %+v, %v", result, err)
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
