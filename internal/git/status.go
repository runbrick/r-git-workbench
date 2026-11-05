package git

import (
	"bytes"
	"context"
	"fmt"
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
	Root   string
	Branch string
	Head   string
	Files  []File
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
	return status, nil
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
