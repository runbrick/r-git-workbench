package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"r-git-workbench/internal/gitexec"
)

type Version struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

type Capabilities struct {
	CanWrite bool
}

func (v Version) Capabilities() Capabilities {
	return Capabilities{CanWrite: v.Major > 2 || v.Major == 2 && v.Minor >= 34}
}

func ParseVersion(output string) (Version, error) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 3 || fields[0] != "git" || fields[1] != "version" {
		return Version{}, fmt.Errorf("unrecognized git version: %q", output)
	}
	parts := strings.SplitN(fields[2], ".", 4)
	if len(parts) < 2 {
		return Version{}, fmt.Errorf("unrecognized git version: %q", output)
	}
	var v Version
	var err error
	v.Major, err = strconv.Atoi(parts[0])
	if err == nil {
		v.Minor, err = strconv.Atoi(parts[1])
	}
	if err == nil && len(parts) > 2 {
		v.Patch, err = strconv.Atoi(parts[2])
	}
	if err != nil || v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
		return Version{}, fmt.Errorf("unrecognized git version: %q", output)
	}
	v.Raw = fields[2]
	return v, nil
}

type Installation struct {
	Path    string
	Version Version
}

func Detect(ctx context.Context, configured string) (Installation, error) {
	var candidates []string
	if configured != "" {
		candidates = append(candidates, configured)
	}
	if path, err := exec.LookPath("git"); err == nil {
		candidates = append(candidates, path)
	}
	switch runtime.GOOS {
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
			if root := os.Getenv(env); root != "" {
				candidates = append(candidates, filepath.Join(root, "Git", "cmd", "git.exe"))
			}
		}
	case "darwin":
		candidates = append(candidates, "/usr/bin/git", "/opt/homebrew/bin/git", "/usr/local/bin/git")
	case "linux":
		candidates = append(candidates, "/usr/bin/git", "/usr/local/bin/git")
	}
	var failures []error
	seen := make(map[string]bool, len(candidates))
	for _, path := range candidates {
		if seen[path] {
			continue
		}
		seen[path] = true
		result, err := (gitexec.Runner{Path: path}).Run(ctx, gitexec.Command{Args: []string{"version"}, Timeout: 5 * time.Second})
		if err == nil {
			var version Version
			version, err = ParseVersion(string(result.Stdout))
			if err == nil {
				return Installation{Path: path, Version: version}, nil
			}
		}
		failures = append(failures, fmt.Errorf("%s: %w", path, err))
		if ctx.Err() != nil {
			return Installation{}, ctx.Err()
		}
	}
	if len(failures) == 0 {
		return Installation{}, errors.New("git executable not found")
	}
	return Installation{}, errors.Join(failures...)
}
