package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"r-git-workbench/internal/gitexec"
)

type File struct {
	Path      string
	Original  string
	Index     byte
	Worktree  byte
	Untracked bool
	Conflict  bool
}

type Status struct {
	Root      string
	Branch    string
	Head      string
	Ahead     int
	Behind    int
	HasTrack  bool
	Files     []File
	Operation OperationState
}

type OperationKind uint8

const (
	OperationNone OperationKind = iota
	OperationMerge
	OperationRebase
	OperationCherryPick
	OperationRevert
)

type OperationState struct {
	Kind      OperationKind
	Target    string
	Current   int
	Remaining int
}

func ReadIgnored(ctx context.Context, runner gitexec.Runner, directory string) ([]string, error) {
	result, err := runner.Run(ctx, gitexec.Command{Dir: directory, Args: []string{"ls-files", "--others", "--ignored", "--exclude-standard", "-z"}, OutputLimit: 8 << 20})
	if err != nil {
		return nil, fmt.Errorf("list ignored files: %w: %s", err, result.Stderr)
	}
	var paths []string
	for _, item := range bytes.Split(result.Stdout, []byte{0}) {
		if len(item) > 0 {
			paths = append(paths, string(item))
		}
	}
	return paths, nil
}

func ReadStatus(ctx context.Context, runner gitexec.Runner, directory string) (Status, error) {
	root, err := runner.Run(ctx, gitexec.Command{Dir: directory, Args: []string{"rev-parse", "--show-toplevel"}})
	if err != nil {
		return Status{}, fmt.Errorf("locate repository: %w: %s", err, root.Stderr)
	}
	path := strings.TrimSuffix(strings.TrimSuffix(string(root.Stdout), "\n"), "\r")
	result, err := runner.Run(ctx, gitexec.Command{Dir: path, Args: []string{"status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all"}})
	if err != nil {
		return Status{}, fmt.Errorf("read status: %w: %s", err, result.Stderr)
	}
	status, err := ParseStatus(result.Stdout)
	if err != nil {
		return Status{}, err
	}
	status.Root = path
	status.Operation, err = ReadOperationState(ctx, runner, path)
	if err != nil {
		return Status{}, err
	}
	return status, nil
}

func ReadOperationState(ctx context.Context, runner gitexec.Runner, directory string) (OperationState, error) {
	markers := []struct {
		kind OperationKind
		name string
	}{
		{OperationRebase, "rebase-merge"},
		{OperationRebase, "rebase-apply"},
		{OperationMerge, "MERGE_HEAD"},
		{OperationCherryPick, "CHERRY_PICK_HEAD"},
		{OperationRevert, "REVERT_HEAD"},
	}
	for _, marker := range markers {
		result, err := runner.Run(ctx, gitexec.Command{Dir: directory, Args: []string{"rev-parse", "--git-path", marker.name}})
		if err != nil {
			return OperationState{}, fmt.Errorf("locate Git operation state: %w: %s", err, result.Stderr)
		}
		name := strings.TrimSpace(string(result.Stdout))
		if !filepath.IsAbs(name) {
			name = filepath.Join(directory, filepath.FromSlash(name))
		}
		info, err := os.Stat(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return OperationState{}, fmt.Errorf("read Git operation state %s: %w", marker.name, err)
		}
		state := OperationState{Kind: marker.kind}
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(name)
			if err != nil {
				return OperationState{}, fmt.Errorf("read Git operation target %s: %w", marker.name, err)
			}
			state.Target = strings.TrimSpace(string(data))
		}
		if marker.kind == OperationRebase {
			state.Current, state.Remaining = readRebaseProgress(name)
		}
		return state, nil
	}
	return OperationState{}, nil
}

func readRebaseProgress(marker string) (int, int) {
	root := marker
	if filepath.Base(root) != "rebase-merge" && filepath.Base(root) != "rebase-apply" {
		return 0, 0
	}
	done := 0
	if data, err := os.ReadFile(filepath.Join(root, "done")); err == nil {
		done = len(bytes.Split(bytes.TrimSpace(data), []byte{'\n'}))
		if len(bytes.TrimSpace(data)) == 0 {
			done = 0
		}
	}
	remaining := 0
	if data, err := os.ReadFile(filepath.Join(root, "git-rebase-todo")); err == nil {
		for _, line := range bytes.Split(data, []byte{'\n'}) {
			line = bytes.TrimSpace(line)
			if len(line) > 0 && line[0] != '#' {
				remaining++
			}
		}
	}
	return done + 1, remaining
}

func ParseStatus(data []byte) (Status, error) {
	var status Status
	for len(data) > 0 && data[0] == '#' {
		end := bytes.IndexAny(data, "\n\x00")
		if end < 0 {
			return Status{}, fmt.Errorf("unterminated status header")
		}
		line := string(data[:end])
		if strings.HasPrefix(line, "# branch.head ") {
			status.Branch = strings.TrimPrefix(line, "# branch.head ")
		}
		if strings.HasPrefix(line, "# branch.oid ") {
			status.Head = strings.TrimPrefix(line, "# branch.oid ")
		}
		if strings.HasPrefix(line, "# branch.ab ") {
			fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(fields) != 2 || !strings.HasPrefix(fields[0], "+") || !strings.HasPrefix(fields[1], "-") {
				return Status{}, fmt.Errorf("invalid branch ahead/behind header %q", line)
			}
			ahead, errAhead := strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
			behind, errBehind := strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
			if errAhead != nil || errBehind != nil || ahead < 0 || behind < 0 {
				return Status{}, fmt.Errorf("invalid branch ahead/behind header %q", line)
			}
			status.Ahead, status.Behind, status.HasTrack = ahead, behind, true
		}
		data = data[end+1:]
	}
	fields := bytes.Split(data, []byte{0})
	for i := 0; i < len(fields); i++ {
		item := string(fields[i])
		if item == "" {
			continue
		}
		if len(item) < 3 {
			return Status{}, fmt.Errorf("invalid status entry %q", item)
		}
		file := File{}
		switch item[:2] {
		case "? ":
			file.Path, file.Untracked = item[2:], true
		case "1 ", "2 ", "u ":
			count := 9
			if item[0] == '2' {
				count = 10
			}
			if item[0] == 'u' {
				count = 11
			}
			parts := strings.SplitN(item, " ", count)
			if len(parts) != count || len(parts[1]) != 2 {
				return Status{}, fmt.Errorf("invalid status entry %q", item)
			}
			file.Index, file.Worktree, file.Path = parts[1][0], parts[1][1], parts[count-1]
			file.Conflict = item[0] == 'u'
			if item[0] == '2' {
				i++
				if i >= len(fields) {
					return Status{}, fmt.Errorf("missing original path for %q", file.Path)
				}
				file.Original = string(fields[i])
			}
		default:
			return Status{}, fmt.Errorf("unknown status entry %q", item)
		}
		if file.Path == "" {
			return Status{}, fmt.Errorf("empty status path")
		}
		status.Files = append(status.Files, file)
	}
	return status, nil
}
