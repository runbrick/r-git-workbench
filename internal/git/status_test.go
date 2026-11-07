package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"r-git-workbench/internal/gitexec"
)

func TestReadOperationState(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	command := exec.Command(gitPath, "init")
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	runner := gitexec.Runner{Path: gitPath}
	gitPathResult, err := runner.Run(context.Background(), gitexec.Command{Dir: dir, Args: []string{"rev-parse", "--git-path", "MERGE_HEAD"}})
	if err != nil {
		t.Fatal(err)
	}
	marker := strings.TrimSpace(string(gitPathResult.Stdout))
	if !filepath.IsAbs(marker) {
		marker = filepath.Join(dir, filepath.FromSlash(marker))
	}
	if err := os.WriteFile(marker, []byte("merge-target\n"), 0600); err != nil {
		t.Fatal(err)
	}
	state, err := ReadOperationState(context.Background(), runner, dir)
	if err != nil || state.Kind != OperationMerge || state.Target != "merge-target" {
		t.Fatalf("merge operation: %+v, %v", state, err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	rebasePath, err := runner.Run(context.Background(), gitexec.Command{Dir: dir, Args: []string{"rev-parse", "--git-path", "rebase-merge"}})
	if err != nil {
		t.Fatal(err)
	}
	rebaseDir := strings.TrimSpace(string(rebasePath.Stdout))
	if !filepath.IsAbs(rebaseDir) {
		rebaseDir = filepath.Join(dir, filepath.FromSlash(rebaseDir))
	}
	if err := os.Mkdir(rebaseDir, 0700); err != nil {
		t.Fatal(err)
	}
	state, err = ReadOperationState(context.Background(), runner, dir)
	if err != nil || state.Kind != OperationRebase {
		t.Fatalf("rebase operation: %+v, %v", state, err)
	}
	if err := os.WriteFile(filepath.Join(rebaseDir, "done"), []byte("pick abc\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rebaseDir, "git-rebase-todo"), []byte("pick def\npick ghi\n# comment\n"), 0600); err != nil {
		t.Fatal(err)
	}
	state, err = ReadOperationState(context.Background(), runner, dir)
	if err != nil || state.Current != 2 || state.Remaining != 2 {
		t.Fatalf("rebase progress: %+v, %v", state, err)
	}
}

func TestParseStatusBranchesRenamesAndConflicts(t *testing.T) {
	data := strings.Join([]string{
		"# branch.oid 0123456789abcdef\n# branch.head feature/test\n# branch.ab +3 -2\n1 .M N... 100644 100644 100644 abc def path with spaces.txt",
		"2 R. N... 100644 100644 100644 abc def R100 renamed.txt",
		"previous.txt",
		"u UU N... 100644 100644 100644 100644 abc def ghi conflict.txt",
		"? untracked\nname.txt",
		"",
	}, "\x00")
	status, err := ParseStatus([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if status.Branch != "feature/test" || status.Head != "0123456789abcdef" || status.Ahead != 3 || status.Behind != 2 || !status.HasTrack || len(status.Files) != 4 {
		t.Fatalf("status: %+v", status)
	}
	if status.Files[0].Path != "path with spaces.txt" || status.Files[0].Worktree != 'M' {
		t.Fatalf("tracked: %+v", status.Files[0])
	}
	if status.Files[1].Path != "renamed.txt" || status.Files[1].Original != "previous.txt" {
		t.Fatalf("renamed: %+v", status.Files[1])
	}
	if !status.Files[2].Conflict || status.Files[2].Path != "conflict.txt" {
		t.Fatalf("conflict: %+v", status.Files[2])
	}
	if !status.Files[3].Untracked || status.Files[3].Path != "untracked\nname.txt" {
		t.Fatalf("untracked: %+v", status.Files[3])
	}
}

func TestParseStatusRejectsInvalidAheadBehind(t *testing.T) {
	for _, header := range []string{"# branch.ab +x -2", "# branch.ab +1 2", "# branch.ab +-1 -0"} {
		if _, err := ParseStatus([]byte(header + "\x00")); err == nil {
			t.Errorf("accepted invalid header %q", header)
		}
	}
}
