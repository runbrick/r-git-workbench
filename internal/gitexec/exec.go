package gitexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Command struct {
	Args    []string
	Dir     string
	Stdin   io.Reader
	Write   bool
	Timeout time.Duration
}

type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type Runner struct {
	Path string
}

func (runner Runner) Run(ctx context.Context, command Command) (Result, error) {
	if runner.Path == "" {
		return Result{}, errors.New("git executable path is empty")
	}
	if len(command.Args) == 0 {
		return Result{}, errors.New("git arguments are empty")
	}
	if command.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, command.Timeout)
		defer cancel()
	}
	args := make([]string, 0, len(command.Args)+4)
	args = append(args, "-c", "core.quotepath=false", "-c", "color.ui=false")
	args = append(args, command.Args...)
	process := exec.CommandContext(ctx, runner.Path, args...)
	process.Dir = command.Dir
	process.Stdin = command.Stdin
	process.Env = overrideEnv(os.Environ(), command.Write)
	var stdout, stderr bytes.Buffer
	process.Stdout = &stdout
	process.Stderr = &stderr
	err := process.Run()
	result := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err != nil {
		result.ExitCode = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.ExitCode = exit.ExitCode()
		}
		return result, fmt.Errorf("git %s: %w", command.Args[0], err)
	}
	return result, nil
}

func overrideEnv(existing []string, write bool) []string {
	keys := []string{"LC_ALL", "LANGUAGE", "GIT_TERMINAL_PROMPT", "GIT_OPTIONAL_LOCKS"}
	result := make([]string, 0, len(existing)+len(keys))
	for _, entry := range existing {
		name, _, _ := strings.Cut(entry, "=")
		remove := false
		for _, key := range keys {
			if strings.EqualFold(name, key) {
				remove = true
				break
			}
		}
		if !remove {
			result = append(result, entry)
		}
	}
	result = append(result, "LC_ALL=C", "LANGUAGE=", "GIT_TERMINAL_PROMPT=0")
	if !write {
		result = append(result, "GIT_OPTIONAL_LOCKS=0")
	}
	return result
}
