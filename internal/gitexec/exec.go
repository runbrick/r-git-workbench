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
	"sync"
	"time"
)

type Command struct {
	Args    []string
	Dir     string
	Stdin   io.Reader
	Write   bool
	Timeout time.Duration
	// OutputLimit bounds captured stdout. Zero uses the default 64 MiB limit.
	OutputLimit int
}

type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type Runner struct {
	Path string
}

const (
	defaultOutputLimit = 64 << 20
	stderrLimit        = 1 << 20
)

// limitedOutput drains all process output while retaining only the first limit bytes.
// Draining prevents Git from blocking on a full pipe when output is truncated.
type limitedOutput struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (output *limitedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := output.limit - output.buffer.Len()
	if remaining < n {
		output.truncated = true
		p = p[:max(remaining, 0)]
	}
	_, _ = output.buffer.Write(p)
	return n, nil
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
	process := exec.Command(runner.Path, args...)
	configureProcess(process)
	process.Dir = command.Dir
	process.Stdin = command.Stdin
	process.Env = overrideEnv(os.Environ(), command.Write)
	if err := ctx.Err(); err != nil {
		return Result{ExitCode: -1}, err
	}
	limit := command.OutputLimit
	if limit <= 0 {
		limit = defaultOutputLimit
	}
	stdout, stderr := limitedOutput{limit: limit}, limitedOutput{limit: stderrLimit}
	process.Stdout = &stdout
	process.Stderr = &stderr
	err := process.Start()
	if err == nil {
		done := make(chan struct{})
		stopped := make(chan struct{})
		var state sync.Mutex
		finished := false
		go func() {
			defer close(stopped)
			select {
			case <-ctx.Done():
				state.Lock()
				if !finished {
					terminateProcess(process)
				}
				state.Unlock()
			case <-done:
			}
		}()
		err = process.Wait()
		state.Lock()
		finished = true
		state.Unlock()
		close(done)
		<-stopped
	}
	result := Result{Stdout: stdout.buffer.Bytes(), Stderr: stderr.buffer.Bytes()}
	if err != nil {
		result.ExitCode = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.ExitCode = exit.ExitCode()
		}
		if stdout.truncated || stderr.truncated {
			return result, fmt.Errorf("git %s output exceeded capture limit (stdout %d, stderr %d bytes): %w", command.Args[0], limit, stderrLimit, err)
		}
		return result, fmt.Errorf("git %s: %w", command.Args[0], err)
	}
	if stdout.truncated || stderr.truncated {
		return result, fmt.Errorf("git %s output exceeded capture limit (stdout %d, stderr %d bytes)", command.Args[0], limit, stderrLimit)
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
