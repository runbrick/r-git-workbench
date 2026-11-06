package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStageUnstageCommitAndStatus(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	command := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	command("init")
	command("config", "user.name", "Test")
	command("config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("one\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, status, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Files) != 1 || !status.Files[0].Untracked {
		t.Fatalf("untracked status: %+v", status)
	}
	patch, err := r.Diff(context.Background(), "new.txt", false)
	if err != nil || !strings.Contains(patch, "+one") {
		t.Fatalf("untracked diff: %q, %v", patch, err)
	}
	comparison, err := r.Compare(context.Background(), status.Files[0], false)
	if err != nil || comparison.Binary || comparison.Left != "" || comparison.Right != "one\n" {
		t.Fatalf("untracked comparison: %+v, %v", comparison, err)
	}
	if err := r.Stage(context.Background(), "new.txt"); err != nil {
		t.Fatal(err)
	}
	status, err = r.Status(context.Background())
	if err != nil || len(status.Files) != 1 || status.Files[0].Index != 'A' {
		t.Fatalf("staged status: %+v, %v", status, err)
	}
	comparison, err = r.Compare(context.Background(), status.Files[0], true)
	if err != nil || comparison.Binary || comparison.Left != "" || comparison.Right != "one\n" {
		t.Fatalf("staged addition comparison: %+v, %v", comparison, err)
	}
	if err := r.Unstage(context.Background(), "new.txt"); err != nil {
		t.Fatal(err)
	}
	status, err = r.Status(context.Background())
	if err != nil || len(status.Files) != 1 || !status.Files[0].Untracked {
		t.Fatalf("unstaged unborn HEAD: %+v, %v", status, err)
	}
	if err := r.Stage(context.Background(), "new.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit(context.Background(), "initial"); err != nil {
		t.Fatal(err)
	}
	commits, err := r.Log(context.Background(), 0, 1)
	if err != nil || len(commits) != 1 || commits[0].Subject != "initial" {
		t.Fatalf("first log page: %+v, %v", commits, err)
	}
	message, paths, err := r.CommitDetails(context.Background(), commits[0].SHA)
	if err != nil || !strings.Contains(message, "initial") || len(paths) != 1 || paths[0] != "new.txt" {
		t.Fatalf("commit details: %q, %+v, %v", message, paths, err)
	}
	fileDiff, err := r.CommitFile(context.Background(), commits[0].SHA, "new.txt")
	if err != nil || fileDiff.Left != "" || fileDiff.Right != "one\n" {
		t.Fatalf("root commit file: %+v, %v", fileDiff, err)
	}
	commits, err = r.Log(context.Background(), 1, 1)
	if err != nil || len(commits) != 0 {
		t.Fatalf("second log page: %+v, %v", commits, err)
	}
	if got := command("log", "-1", "--format=%s"); got != "initial" {
		t.Fatalf("commit: %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("two\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, err = r.Status(context.Background())
	if err != nil || len(status.Files) != 1 || status.Files[0].Worktree != 'M' {
		t.Fatalf("working status: %+v, %v", status, err)
	}
	comparison, err = r.Compare(context.Background(), status.Files[0], false)
	if err != nil || comparison.Left != "one\n" || comparison.Right != "two\n" {
		t.Fatalf("working comparison: %+v, %v", comparison, err)
	}
	patch, err = r.Diff(context.Background(), "new.txt", false)
	if err != nil || !strings.Contains(patch, "+two") {
		t.Fatalf("diff: %q, %v", patch, err)
	}
	if err := r.Stage(context.Background(), "new.txt"); err != nil {
		t.Fatal(err)
	}
	patch, err = r.Diff(context.Background(), "new.txt", true)
	if err != nil || !strings.Contains(patch, "+two") {
		t.Fatalf("staged diff: %q, %v", patch, err)
	}
	status, err = r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	comparison, err = r.Compare(context.Background(), status.Files[0], true)
	if err != nil || comparison.Left != "one\n" || comparison.Right != "two\n" {
		t.Fatalf("staged comparison: %+v, %v", comparison, err)
	}
	if err := r.Commit(context.Background(), "second"); err != nil {
		t.Fatal(err)
	}
	commits, err = r.Log(context.Background(), 0, 1)
	if err != nil || len(commits) != 1 {
		t.Fatalf("second commit log: %+v, %v", commits, err)
	}
	fileDiff, err = r.CommitFile(context.Background(), commits[0].SHA, "new.txt")
	if err != nil || fileDiff.Left != "one\n" || fileDiff.Right != "two\n" {
		t.Fatalf("changed commit file: %+v, %v", fileDiff, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("three\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.Stage(context.Background(), "new.txt"); err != nil {
		t.Fatal(err)
	}
	odd := " leading name.txt"
	if err := os.WriteFile(filepath.Join(dir, odd), []byte("odd\n"), 0600); err != nil {
		t.Fatal(err)
	}
	newline := "two\nlines.txt"
	if runtime.GOOS != "windows" {
		if err := os.WriteFile(filepath.Join(dir, newline), []byte("newline\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	status, err = r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found, foundNewline := false, false
	for _, file := range status.Files {
		if file.Path == odd && file.Untracked {
			found = true
		}
		if file.Path == newline && file.Untracked {
			foundNewline = true
		}
	}
	if !found || runtime.GOOS != "windows" && !foundNewline {
		t.Fatalf("special paths missing: %+v", status.Files)
	}
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho rejected >&2\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit(context.Background(), "keep this message"); err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("hook failure: %v", err)
	}
	if got := command("log", "-1", "--format=%s"); got != "second" {
		t.Fatalf("hook changed HEAD: %q", got)
	}
	status, err = r.Status(context.Background())
	wantFiles := 2
	if runtime.GOOS != "windows" {
		wantFiles = 3
	}
	if err != nil || len(status.Files) != wantFiles || status.Files[0].Index != 'M' {
		t.Fatalf("hook changed staged files: %+v, %v", status, err)
	}
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	command("commit", "-m", "third")
	command("rm", "new.txt")
	command("commit", "-m", "delete file")
	deletedSHA := command("rev-parse", "HEAD")
	fileDiff, err = r.CommitFile(context.Background(), deletedSHA, "new.txt")
	if err != nil || fileDiff.Left != "three\n" || fileDiff.Right != "" || !strings.Contains(fileDiff.Patch, "-three") {
		t.Fatalf("deleted commit file: %+v, %v", fileDiff, err)
	}
	if _, err := r.CommitFile(context.Background(), "invalid", "new.txt"); err == nil {
		t.Fatal("accepted invalid SHA")
	}
}

func TestCompareLargeFileFallsBackToPatch(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	cmd := exec.Command(gitPath, "init", dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(dir, "large.bin"), make([]byte, 513<<10), 0600); err != nil {
		t.Fatal(err)
	}
	r, status, err := Open(context.Background(), gitPath, dir)
	if err != nil || len(status.Files) != 1 {
		t.Fatalf("open: %+v, %v", status, err)
	}
	emptyLog, err := r.Log(context.Background(), 0, 100)
	if err != nil || len(emptyLog) != 0 {
		t.Fatalf("empty repository log: %+v, %v", emptyLog, err)
	}
	comparison, err := r.Compare(context.Background(), status.Files[0], false)
	if err != nil || !comparison.Binary {
		t.Fatalf("large untracked comparison: %+v, %v", comparison, err)
	}
	if err := r.Stage(context.Background(), "large.bin"); err != nil {
		t.Fatal(err)
	}
	status, err = r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	comparison, err = r.Compare(context.Background(), status.Files[0], true)
	if err != nil || !comparison.Binary {
		t.Fatalf("large staged comparison: %+v, %v", comparison, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "small.bin"), []byte{0, 1, 2}, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty.txt"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	status, err = r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range status.Files {
		if !file.Untracked {
			continue
		}
		comparison, err := r.Compare(context.Background(), file, false)
		if err != nil {
			t.Fatalf("compare %q: %v", file.Path, err)
		}
		if file.Path == "small.bin" && !comparison.Binary {
			t.Fatalf("binary file treated as text: %+v", comparison)
		}
		if file.Path == "small.bin" && comparison.Patch == "" {
			t.Fatalf("binary file has no fallback detail: %+v", comparison)
		}
		if file.Path == "empty.txt" && (comparison.Binary || comparison.Left != "" || comparison.Right != "") {
			t.Fatalf("empty file comparison: %+v", comparison)
		}
	}
}

func TestLogPages(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	run("init")
	run("config", "user.name", "Log Test")
	run("config", "user.email", "log@example.com")
	for i := 0; i < 105; i++ {
		run("commit", "--allow-empty", "-m", fmt.Sprintf("entry %03d", i))
	}
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.Log(context.Background(), 0, 100)
	if err != nil || len(first) != 100 || first[0].Subject != "entry 104" {
		t.Fatalf("first page: %d, %v", len(first), err)
	}
	second, err := r.Log(context.Background(), 100, 100)
	if err != nil || len(second) != 5 || second[4].Subject != "entry 000" {
		t.Fatalf("second page: %d, %v", len(second), err)
	}
	if first[99].SHA == second[0].SHA {
		t.Fatal("pagination repeated a commit")
	}
	filtered, err := r.LogFiltered(context.Background(), 0, 2, "ENTRY 10")
	if err != nil || len(filtered) != 2 || filtered[0].Subject != "entry 104" || filtered[1].Subject != "entry 103" {
		t.Fatalf("filtered first page: %+v, %v", filtered, err)
	}
	filtered, err = r.LogFiltered(context.Background(), 2, 2, "ENTRY 10")
	if err != nil || len(filtered) != 2 || filtered[0].Subject != "entry 102" || filtered[1].Subject != "entry 101" {
		t.Fatalf("filtered second page: %+v, %v", filtered, err)
	}
	filtered, err = r.LogFiltered(context.Background(), 0, 2, "entry [")
	if err != nil || len(filtered) != 0 {
		t.Fatalf("literal filter: %+v, %v", filtered, err)
	}
	filtered, err = r.LogQuery(context.Background(), 0, 2, LogFilter{Author: "LOG TEST"})
	if err != nil || len(filtered) != 2 || filtered[0].Subject != "entry 104" {
		t.Fatalf("author filter: %+v, %v", filtered, err)
	}
	filtered, err = r.LogQuery(context.Background(), 0, 2, LogFilter{Path: "../outside"})
	if err == nil || len(filtered) != 0 {
		t.Fatalf("unsafe path filter: %+v, %v", filtered, err)
	}
}

func TestMergeCommitDetailsAndFileDiff(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.name", "Merge Test")
	run("config", "user.email", "merge@example.com")
	if err := os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("initial\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "shared.txt")
	run("commit", "-m", "initial")
	base := run("branch", "--show-current")
	run("switch", "-c", "feature")
	if err := os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("feature\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("commit", "-am", "feature edit")
	run("switch", base)
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "base.txt")
	run("commit", "-m", "base edit")
	run("merge", "--no-ff", "-m", "merge feature", "feature")
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	sha := run("rev-parse", "HEAD")
	message, files, err := r.CommitDetails(context.Background(), sha)
	if err != nil || !strings.Contains(message, "merge feature") || len(files) != 1 || files[0] != "shared.txt" {
		t.Fatalf("merge details: %q, %+v, %v", message, files, err)
	}
	diff, err := r.CommitFile(context.Background(), sha, "shared.txt")
	if err != nil || diff.Left != "initial\n" || diff.Right != "feature\n" {
		t.Fatalf("merge file comparison: %+v, %v", diff, err)
	}
}

func TestRefsListsLocalAndRemoteBranches(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.name", "Refs Test")
	run("config", "user.email", "refs@example.com")
	run("commit", "--allow-empty", "-m", "initial")
	run("branch", "feature/test")
	run("config", "remote.origin.url", "../unused-remote.git")
	run("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	run("update-ref", "refs/remotes/origin/feature/test", run("rev-parse", "HEAD"))
	run("branch", "--set-upstream-to=origin/feature/test", "feature/test")
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := r.Refs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	local, remote := false, false
	for _, ref := range refs {
		switch {
		case ref.Name == "feature/test" && !ref.Remote:
			local = strings.HasSuffix(ref.Upstream, "origin/feature/test") && len(ref.SHA) == 40
		case ref.Name == "origin/feature/test" && ref.Remote:
			remote = len(ref.SHA) == 40
		}
	}
	if !local || !remote {
		t.Fatalf("missing local/upstream or remote ref: %+v", refs)
	}
	mainName := run("branch", "--show-current")
	run("switch", "feature/test")
	run("commit", "--allow-empty", "-m", "only feature")
	mainLog, err := r.LogQuery(context.Background(), 0, 10, LogFilter{Ref: "refs/heads/" + mainName})
	if err != nil || len(mainLog) != 1 || mainLog[0].Subject != "initial" {
		t.Fatalf("main branch log: %+v, %v", mainLog, err)
	}
	featureLog, err := r.LogQuery(context.Background(), 0, 10, LogFilter{Ref: "refs/heads/feature/test"})
	if err != nil || len(featureLog) != 2 || featureLog[0].Subject != "only feature" {
		t.Fatalf("feature branch log: %+v, %v", featureLog, err)
	}
	if _, err := r.LogQuery(context.Background(), 0, 10, LogFilter{Ref: "--all"}); err == nil {
		t.Fatal("accepted arbitrary Git option as branch reference")
	}
}

func TestCreateBranchValidatesNameAndPreservesHead(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.name", "Branch Test")
	run("config", "user.email", "branch@example.com")
	run("commit", "--allow-empty", "-m", "initial")
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	head := run("rev-parse", "HEAD")
	if err := r.CreateBranch(context.Background(), "feature/test"); err != nil {
		t.Fatal(err)
	}
	if got := run("rev-parse", "refs/heads/feature/test"); got != head {
		t.Fatalf("new branch points to %q, want %q", got, head)
	}
	for _, name := range []string{"feature/test", "bad name", "--help", "bad\nname", ""} {
		if err := r.CreateBranch(context.Background(), name); err == nil {
			t.Errorf("accepted duplicate or invalid branch name %q", name)
		}
	}
	if got := run("branch", "--show-current"); got == "feature/test" {
		t.Fatal("creating a branch switched the working tree")
	}
	if got := run("rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD changed from %q to %q", head, got)
	}
}

func TestSwitchBranchPreservesDirtyWorktreeOnRefusal(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.name", "Switch Test")
	run("config", "user.email", "switch@example.com")
	file := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(file, []byte("initial\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "tracked.txt")
	run("commit", "-m", "initial")
	main := run("branch", "--show-current")
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.CreateBranch(context.Background(), "feature/test"); err != nil {
		t.Fatal(err)
	}
	if err := r.SwitchBranch(context.Background(), "feature/test"); err != nil {
		t.Fatalf("switch to feature: %v", err)
	}
	if got := run("branch", "--show-current"); got != "feature/test" {
		t.Fatalf("current branch: %q", got)
	}
	if err := os.WriteFile(file, []byte("feature\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "tracked.txt")
	run("commit", "-m", "feature edit")
	if err := r.SwitchBranch(context.Background(), main); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("local change\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.SwitchBranch(context.Background(), "feature/test"); err == nil {
		t.Fatal("overwriting dirty file was permitted")
	}
	if got := run("branch", "--show-current"); got != main {
		t.Fatalf("failed switch changed branch to %q", got)
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "local change\n" {
		t.Fatalf("failed switch lost worktree contents: %q, %v", data, err)
	}
	for _, name := range []string{"bad name", "--help", "origin/unknown"} {
		if err := r.SwitchBranch(context.Background(), name); err == nil {
			t.Errorf("switched to invalid or missing local branch %q", name)
		}
	}
}

func TestBranchManagementAndDeletePreflight(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.name", "Branch Manager")
	run("config", "user.email", "branch-manager@example.com")
	run("commit", "--allow-empty", "-m", "initial")
	run("branch", "feature/delete-me")
	run("config", "remote.origin.url", "../unused.git")
	run("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	run("update-ref", "refs/remotes/origin/feature/delete-me", run("rev-parse", "HEAD"))
	run("branch", "--set-upstream-to=origin/feature/delete-me", "feature/delete-me")
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.RenameBranch(context.Background(), "feature/delete-me", "feature/renamed"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetUpstream(context.Background(), "feature/renamed", ""); err != nil {
		t.Fatal(err)
	}
	if got := run("for-each-ref", "--format=%(upstream)", "refs/heads/feature/renamed"); got != "" {
		t.Fatalf("upstream was not removed: %q", got)
	}
	if err := r.SetUpstream(context.Background(), "feature/renamed", "refs/remotes/origin/feature/delete-me"); err != nil {
		t.Fatal(err)
	}
	if got := run("for-each-ref", "--format=%(upstream)", "refs/heads/feature/renamed"); got != "refs/remotes/origin/feature/delete-me" {
		t.Fatalf("wrong upstream: %q", got)
	}
	if err := r.CheckoutRemoteBranch(context.Background(), "refs/remotes/origin/feature/delete-me", "feature/tracking"); err != nil {
		t.Fatal(err)
	}
	if got := run("for-each-ref", "--format=%(upstream)", "refs/heads/feature/tracking"); got != "refs/remotes/origin/feature/delete-me" {
		t.Fatalf("tracking branch upstream: %q", got)
	}
	plan, err := r.PlanDeleteBranch(context.Background(), "feature/renamed")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Target != run("rev-parse", "refs/heads/feature/renamed") {
		t.Fatalf("delete preview target %s does not match branch", plan.Target)
	}
	if err := r.DeleteBranch(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PlanDeleteBranch(context.Background(), "feature/tracking"); err == nil {
		t.Fatal("current branch was offered for deletion")
	}
	if _, err := r.PlanDeleteBranch(context.Background(), "missing"); err == nil {
		t.Fatal("missing branch was offered for deletion")
	}
}

func TestDeleteBranchRejectsStalePreflight(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.name", "Stale Plan")
	run("config", "user.email", "stale@example.com")
	run("commit", "--allow-empty", "-m", "initial")
	run("branch", "feature/stale")
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := r.PlanDeleteBranch(context.Background(), "feature/stale")
	if err != nil {
		t.Fatal(err)
	}
	run("commit", "--allow-empty", "-m", "advance head")
	if err := r.DeleteBranch(context.Background(), plan); err == nil {
		t.Fatal("stale delete preflight was accepted")
	}
	if got := run("show-ref", "--verify", "refs/heads/feature/stale"); got == "" {
		t.Fatal("branch was deleted after stale preflight")
	}
}

func TestRemoteCheckoutPlansCancelStashAndForce(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(dir, "global"), "GIT_CONFIG_SYSTEM="+filepath.Join(dir, "system"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.name", "Checkout Test")
	run("config", "user.email", "checkout@example.com")
	file := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(file, []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "tracked.txt")
	run("commit", "-m", "base")
	head := run("rev-parse", "HEAD")
	run("branch", "remote-target")
	run("config", "remote.origin.url", "../unused.git")
	run("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	run("update-ref", "refs/remotes/origin/remote-target", head)
	if err := os.WriteFile(file, []byte("uncommitted\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("untracked\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := r.PlanCheckoutRemoteBranch(context.Background(), "refs/remotes/origin/remote-target", "local-stash")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.DirtyFiles) != 2 || plan.DirtyFiles[0] != "new.txt" || plan.DirtyFiles[1] != "tracked.txt" {
		t.Fatalf("dirty files not listed in preview: %+v", plan.DirtyFiles)
	}
	if err := r.ExecuteCheckoutRemoteBranch(context.Background(), plan, CheckoutSafe); err == nil {
		t.Fatal("safe checkout accepted a dirty worktree; cancel must leave it untouched")
	}
	if got, _ := os.ReadFile(file); string(got) != "uncommitted\n" {
		t.Fatalf("cancel changed tracked file: %q", got)
	}
	if err := r.ExecuteCheckoutRemoteBranch(context.Background(), plan, CheckoutStash); err != nil {
		t.Fatal(err)
	}
	if got := run("branch", "--show-current"); got != "local-stash" {
		t.Fatalf("stash checkout branch: %s", got)
	}
	if got := run("stash", "list"); !strings.Contains(got, "before remote branch checkout") {
		t.Fatalf("changes were not stashed: %q", got)
	}
	if got, err := os.ReadFile(file); err != nil || strings.ReplaceAll(string(got), "\r\n", "\n") != "base\n" {
		t.Fatalf("stashed tracked worktree not clean: %q, %v", got, err)
	}
	run("switch", "-")
	if err := os.WriteFile(file, []byte("force me\n"), 0600); err != nil {
		t.Fatal(err)
	}
	forcePlan, err := r.PlanCheckoutRemoteBranch(context.Background(), "refs/remotes/origin/remote-target", "local-force")
	if err != nil {
		t.Fatal(err)
	}
	run("commit", "--allow-empty", "-m", "advance head after preview")
	if err := r.ExecuteCheckoutRemoteBranch(context.Background(), forcePlan, CheckoutForce); err == nil {
		t.Fatal("force checkout accepted a stale preflight")
	}
	if got, _ := os.ReadFile(file); string(got) != "force me\n" {
		t.Fatalf("stale force checkout changed tracked file: %q", got)
	}
	forcePlan, err = r.PlanCheckoutRemoteBranch(context.Background(), "refs/remotes/origin/remote-target", "local-force")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ExecuteCheckoutRemoteBranch(context.Background(), forcePlan, CheckoutForce); err != nil {
		t.Fatal(err)
	}
	if got := run("branch", "--show-current"); got != "local-force" {
		t.Fatalf("force checkout branch: %s", got)
	}
	if got, _ := os.ReadFile(file); strings.ReplaceAll(string(got), "\r\n", "\n") != "base\n" {
		t.Fatalf("force checkout did not replace tracked file: %q", got)
	}
}

func TestRemoteFetchPullPushAgainstBareRepository(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	root := t.TempDir()
	remoteDir := filepath.Join(root, "remote.git")
	localDir := filepath.Join(root, "local")
	otherDir := filepath.Join(root, "other")
	global := filepath.Join(root, "global")
	system := filepath.Join(root, "system")
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+global, "GIT_CONFIG_SYSTEM="+system)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(root, "init", "--bare", remoteDir)
	run(remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")
	if err := os.Mkdir(localDir, 0700); err != nil {
		t.Fatal(err)
	}
	run(localDir, "init")
	run(localDir, "config", "user.name", "Remote Test")
	run(localDir, "config", "user.email", "remote@example.com")
	if err := os.WriteFile(filepath.Join(localDir, "base.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(localDir, "add", "base.txt")
	run(localDir, "commit", "-m", "base")
	run(localDir, "branch", "-M", "main")
	run(localDir, "remote", "add", "origin", remoteDir)
	run(localDir, "push", "-u", "origin", "main")
	r, _, err := Open(context.Background(), gitPath, localDir)
	if err != nil {
		t.Fatal(err)
	}
	var progress []string
	if err := r.Remote(context.Background(), "fetch", func(line string) { progress = append(progress, line) }); err != nil {
		t.Fatal(err)
	}
	status, err := r.Status(context.Background())
	if err != nil || !status.HasTrack || status.Ahead != 0 || status.Behind != 0 {
		t.Fatalf("synced upstream status: %+v, %v", status, err)
	}
	if err := os.Mkdir(otherDir, 0700); err != nil {
		t.Fatal(err)
	}
	run(otherDir, "clone", remoteDir, ".")
	run(otherDir, "config", "user.name", "Remote Writer")
	run(otherDir, "config", "user.email", "writer@example.com")
	if err := os.WriteFile(filepath.Join(otherDir, "remote.txt"), []byte("from remote\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(otherDir, "add", "remote.txt")
	run(otherDir, "commit", "-m", "remote update")
	run(otherDir, "push", "origin", "main")
	if err := r.Remote(context.Background(), "fetch", func(line string) { progress = append(progress, line) }); err != nil {
		t.Fatal(err)
	}
	status, err = r.Status(context.Background())
	if err != nil || !status.HasTrack || status.Ahead != 0 || status.Behind != 1 {
		t.Fatalf("behind upstream status: %+v, %v", status, err)
	}
	if err := r.Remote(context.Background(), "pull", func(line string) { progress = append(progress, line) }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(localDir, "remote.txt")); err != nil {
		t.Fatalf("pull did not update worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "local.txt"), []byte("from local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(localDir, "add", "local.txt")
	run(localDir, "commit", "-m", "local update")
	if err := os.WriteFile(filepath.Join(otherDir, "other.txt"), []byte("diverged remote\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(otherDir, "add", "other.txt")
	run(otherDir, "commit", "-m", "diverged remote update")
	run(otherDir, "push", "origin", "main")
	if err := r.Remote(context.Background(), "fetch", nil); err != nil {
		t.Fatal(err)
	}
	status, err = r.Status(context.Background())
	if err != nil || !status.HasTrack || status.Ahead != 1 || status.Behind != 1 {
		t.Fatalf("diverged upstream status: %+v, %v", status, err)
	}
	if err := r.RemoteWithPolicy(context.Background(), "pull", PullFFOnly, nil); err == nil {
		t.Fatal("fast-forward-only pull accepted diverged history")
	}
	if err := r.RemoteWithPolicy(context.Background(), "pull", PullRebase, nil); err != nil {
		t.Fatalf("rebase pull failed: %v", err)
	}
	pushPlan, err := r.PlanPush(context.Background())
	if err != nil || len(pushPlan.Commits) != 1 || pushPlan.Commits[0].Subject != "local update" {
		t.Fatalf("push preview: %+v, %v", pushPlan, err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "late.txt"), []byte("after preview\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(localDir, "add", "late.txt")
	run(localDir, "commit", "-m", "commit after preview")
	if err := r.ExecutePush(context.Background(), pushPlan, nil); err == nil {
		t.Fatal("stale push preview was executed")
	}
	pushPlan, err = r.PlanPush(context.Background())
	if err != nil || len(pushPlan.Commits) != 2 {
		t.Fatalf("refreshed push preview: %+v, %v", pushPlan, err)
	}
	if err := r.ExecutePush(context.Background(), pushPlan, func(line string) { progress = append(progress, line) }); err != nil {
		t.Fatal(err)
	}
	if got := run(remoteDir, "show", "main:local.txt"); got != "from local" {
		t.Fatalf("push did not update remote: %q", got)
	}
	if got := run(remoteDir, "show", "main:late.txt"); got != "after preview" {
		t.Fatalf("push did not update remote with refreshed plan: %q", got)
	}
	if err := r.RemoteWithPolicy(context.Background(), "pull", PullPolicy(99), nil); err == nil {
		t.Fatal("accepted unknown pull policy")
	}
	if err := r.Remote(context.Background(), "unknown", nil); err == nil {
		t.Fatal("accepted unknown remote operation")
	}
}

func TestForcePushUsesLeaseAndRejectsStalePlan(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	root := t.TempDir()
	remoteDir := filepath.Join(root, "remote.git")
	localDir := filepath.Join(root, "local")
	otherDir := filepath.Join(root, "other")
	global, system := filepath.Join(root, "global"), filepath.Join(root, "system")
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+global, "GIT_CONFIG_SYSTEM="+system)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(root, "init", "--bare", remoteDir)
	run(remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")
	if err := os.Mkdir(localDir, 0700); err != nil {
		t.Fatal(err)
	}
	run(localDir, "init")
	run(localDir, "config", "user.name", "Lease Test")
	run(localDir, "config", "user.email", "lease@example.com")
	if err := os.WriteFile(filepath.Join(localDir, "local.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(localDir, "add", "local.txt")
	run(localDir, "commit", "-m", "base")
	run(localDir, "branch", "-M", "main")
	run(localDir, "remote", "add", "origin", remoteDir)
	run(localDir, "push", "-u", "origin", "main")
	if err := os.Mkdir(otherDir, 0700); err != nil {
		t.Fatal(err)
	}
	run(otherDir, "clone", remoteDir, ".")
	run(otherDir, "config", "user.name", "Other Writer")
	run(otherDir, "config", "user.email", "other@example.com")
	if err := os.WriteFile(filepath.Join(localDir, "local.txt"), []byte("local replacement\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(localDir, "commit", "-am", "local rewrite")
	r, _, err := Open(context.Background(), gitPath, localDir)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := r.PlanForcePush(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, "other.txt"), []byte("remote update\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(otherDir, "add", "other.txt")
	run(otherDir, "commit", "-m", "remote update")
	run(otherDir, "push", "origin", "main")
	if err := r.ExecuteForcePush(context.Background(), plan, nil); err == nil {
		t.Fatal("stale force-push lease succeeded")
	}
	run(localDir, "fetch", "origin")
	plan, err = r.PlanForcePush(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ExecuteForcePush(context.Background(), plan, nil); err != nil {
		t.Fatal(err)
	}
	if got := run(remoteDir, "show", "main:local.txt"); got != "local replacement" {
		t.Fatalf("force push did not update remote: %q", got)
	}
}

func TestLogFiltersPathAndAuthor(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	run("init")
	run("config", "user.name", "Path Author")
	run("config", "user.email", "path@example.com")
	for i, name := range []string{"one.txt", "two.txt", "one.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(fmt.Sprintf("%d\n", i)), 0600); err != nil {
			t.Fatal(err)
		}
		run("add", name)
		run("commit", "-m", fmt.Sprintf("entry %d", i))
	}
	r, _, err := Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	filter := LogFilter{Path: "one.txt", Author: "PATH AUTHOR", Message: "entry"}
	first, err := r.LogQuery(context.Background(), 0, 1, filter)
	if err != nil || len(first) != 1 || first[0].Subject != "entry 2" {
		t.Fatalf("filtered first page: %+v, %v", first, err)
	}
	second, err := r.LogQuery(context.Background(), 1, 1, filter)
	if err != nil || len(second) != 1 || second[0].Subject != "entry 0" {
		t.Fatalf("filtered second page: %+v, %v", second, err)
	}
	missing, err := r.LogQuery(context.Background(), 0, 10, LogFilter{Path: "missing.txt"})
	if err != nil || len(missing) != 0 {
		t.Fatalf("missing path: %+v, %v", missing, err)
	}
	old, err := r.LogQuery(context.Background(), 0, 10, LogFilter{Until: "2000-01-01"})
	if err != nil || len(old) != 0 {
		t.Fatalf("historical date filter: %+v, %v", old, err)
	}
	if _, err := r.LogQuery(context.Background(), 0, 10, LogFilter{Since: "2025-02-30"}); err == nil {
		t.Fatal("accepted invalid date filter")
	}
	if _, err := r.LogQuery(context.Background(), 0, 10, LogFilter{Since: "2025-02-01", Until: "2025-01-01"}); err == nil {
		t.Fatal("accepted inverted date range")
	}
}
