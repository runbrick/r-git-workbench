package shell

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/headless"
	"github.com/vivaui/viva/ui"

	"r-git-workbench/internal/git"
	"r-git-workbench/internal/i18n"
	"r-git-workbench/internal/keymap"
	"r-git-workbench/internal/repo"
)

func TestDriverOpenStageAndCommit(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system Git is unavailable")
	}
	dir := t.TempDir()
	cmd := exec.Command(gitPath, "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %v: %s", err, output)
	}
	for _, args := range [][]string{{"config", "user.name", "Test"}, {"config", "user.email", "test@example.com"}} {
		cmd = exec.Command(gitPath, args...)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("config: %v: %s", err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.txt"), []byte("hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m := New(i18n.English)
	m.SetInitialGit(git.Installation{Path: gitPath, Version: git.Version{Major: 2, Minor: 40}}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.SetContext(ctx)
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	wait := func(check func() bool) {
		t.Helper()
		deadline := time.After(10 * time.Second)
		for !check() {
			select {
			case <-driver.Wakes():
				if err := driver.Pump(); err != nil {
					t.Fatal(err)
				}
			case <-deadline:
				t.Fatalf("timed out; error=%s", m.operationError)
			}
		}
	}
	click := func(id ui.ID) {
		t.Helper()
		for _, hit := range driver.Snapshot().Hits {
			if hit.ID == id {
				if err := driver.Click(hit.Bounds.X+hit.Bounds.Width/2, hit.Bounds.Y+hit.Bounds.Height/2); err != nil {
					t.Fatal(err)
				}
				return
			}
		}
		t.Fatalf("control %s missing", tree.Text(id))
	}
	if err := driver.QueueOpenDirectory(dir, nil); err != nil {
		t.Fatal(err)
	}
	click(browseID)
	if request, ok := driver.LastOpenDirectoryRequest(); !ok || request.Title != m.Language.Text(i18n.OpenRepository) {
		t.Fatalf("directory picker: %+v, %t", request, ok)
	}
	wait(func() bool { return m.repository != nil && !m.busy })
	if len(m.status.Files) != 1 || !m.status.Files[0].Untracked {
		t.Fatalf("opened: %+v", m.status)
	}
	if err := driver.Resize(720, 480); err != nil {
		t.Fatal(err)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("opened narrow layout: %+v", problems)
	}
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(fileScope.At(0))
	wait(func() bool { return m.diff != "" })
	if !m.showComparison || m.comparison.Left != "" || m.comparison.Right != "hello\n" || len(m.aligned.Rows) == 0 {
		t.Fatalf("untracked comparison: %+v", m.comparison)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("diff layout: %+v", problems)
	}
	r := m.diffSlot.Rect()
	if r.Width <= 0 || r.Height <= 0 {
		t.Fatalf("empty comparison bounds: %+v", r)
	}
	if err := driver.Scroll(ui.ScrollEvent{X: r.X + r.Width/2, Y: r.Y + r.Height/2, DeltaY: 32}); err != nil {
		t.Fatal(err)
	}
	if !m.showComparison {
		t.Fatal("comparison disappeared after scroll")
	}
	click(stageID)
	wait(func() bool { return !m.busy })
	if len(m.status.Files) != 1 || m.status.Files[0].Index != 'A' {
		t.Fatalf("staged: %+v; %s", m.status, m.operationError)
	}
	click(fileScope.At(1))
	wait(func() bool { return m.showComparison })
	if m.comparison.Left != "" || m.comparison.Right != "hello\n" {
		t.Fatalf("staged comparison: %+v", m.comparison)
	}
	click(messageID)
	if err := driver.Type("first commit"); err != nil {
		t.Fatal(err)
	}
	if err := driver.Key(ui.KeyEnter, 0); err != nil {
		t.Fatal(err)
	}
	if err := driver.Key(ui.KeyEnter, 0); err != nil {
		t.Fatal(err)
	}
	if err := driver.Type("details"); err != nil {
		t.Fatal(err)
	}
	if m.message.Text() != "first commit\n\ndetails" {
		t.Fatalf("multiline draft: %q", m.message.Text())
	}
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(commitID)
	wait(func() bool { return !m.busy })
	if m.operationError != "" || len(m.status.Files) != 0 || m.message.Text() != "" {
		t.Fatalf("commit: %+v; error=%s; message=%q", m.status, m.operationError, m.message.Text())
	}
	click(ids[1])
	wait(func() bool { return !m.logLoading && len(m.commits) == 1 })
	if m.commits[0].Subject != "first commit" {
		t.Fatalf("log: %+v", m.commits)
	}
	wait(func() bool { return len(m.refs) > 0 })
	click(branchID)
	if m.Active != keymap.ShowBranches {
		t.Fatalf("branch button opened %d", m.Active)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("branch layout: %+v", problems)
	}
	m.branchName.SetText("bad name")
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(createBranchID)
	wait(func() bool { return !m.busy })
	if m.operationError == "" || m.branchName.Text() != "bad name" {
		t.Fatalf("failed branch lost input or error: %q, %q", m.branchName.Text(), m.operationError)
	}
	m.branchName.SetText("feature/created")
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(createBranchID)
	wait(func() bool {
		if m.busy || m.branchName.Text() != "" {
			return false
		}
		for _, ref := range m.refs {
			if !ref.Remote && ref.Name == "feature/created" {
				return true
			}
		}
		return false
	})
	if m.operationError != "" {
		t.Fatalf("created branch with error: %s", m.operationError)
	}
	for i, ref := range m.refs {
		if ref.Name == "feature/created" && !ref.Remote {
			click(branchSwitchScope.At(i))
			break
		}
	}
	wait(func() bool { return !m.busy && m.status.Branch == "feature/created" })
	if m.operationError != "" {
		t.Fatalf("switch branch: %s", m.operationError)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("switched branch layout: %+v", problems)
	}
	if err := driver.Resize(720, 480); err != nil {
		t.Fatal(err)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("narrow branch layout: %+v", problems)
	}
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(branchScope.At(0))
	wait(func() bool { return !m.logLoading && len(m.commits) == 1 })
	if m.Active != keymap.ShowLog || m.logFilter.Ref == "" {
		t.Fatalf("branch log: action=%d, filter=%+v", m.Active, m.logFilter)
	}
	click(filterClearID)
	wait(func() bool { return !m.logLoading && len(m.commits) == 1 })
	if m.logFilter.Ref != "" {
		t.Fatalf("clear kept branch filter: %+v", m.logFilter)
	}
	m.logQuery.SetText("missing message")
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(filterSubmitID)
	wait(func() bool { return !m.logLoading })
	if len(m.commits) != 0 || m.logFilter.Message != "missing message" {
		t.Fatalf("filtered log: %+v, %+v; %s", m.commits, m.logFilter, m.operationError)
	}
	click(filterClearID)
	wait(func() bool { return !m.logLoading && len(m.commits) == 1 })
	if m.logFilter != (repo.LogFilter{}) || m.logQuery.Text() != "" {
		t.Fatalf("cleared filter: %+v, %q", m.logFilter, m.logQuery.Text())
	}
	if err := driver.Resize(720, 480); err != nil {
		t.Fatal(err)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("narrow log layout: %+v", problems)
	}
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(logScope.At(0))
	wait(func() bool { return len(m.commitFiles) == 1 })
	if m.commitFiles[0] != "sample.txt" || m.commitMessage == "" {
		t.Fatalf("details: %q, %+v", m.commitMessage, m.commitFiles)
	}
	click(copySHAID)
	if copied, err := app.ClipboardText(); err != nil || copied != m.selectedCommit {
		t.Fatalf("copied SHA: %q, %v; expected %q", copied, err, m.selectedCommit)
	}
	click(commitFileScope.At(0))
	wait(func() bool { return m.commitFileReady })
	if m.commitFileDiff.Left != "" || m.commitFileDiff.Right != "hello\n" || len(m.commitFileAligned.Rows) == 0 {
		t.Fatalf("commit file comparison: %+v", m.commitFileDiff)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("commit file layout: %+v", problems)
	}
	if err := driver.Resize(720, 480); err != nil {
		t.Fatal(err)
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("narrow commit file layout: %+v", problems)
	}
	if err := driver.Resize(1100, 720); err != nil {
		t.Fatal(err)
	}
	click(ids[0])
	if err := os.WriteFile(filepath.Join(dir, "sample.txt"), []byte("changed outside the app\n"), 0600); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { return !m.busy && len(m.status.Files) == 1 && m.status.Files[0].Worktree == 'M' })
	click(treeModeID)
	if !m.treeMode {
		t.Fatal("tree mode was not enabled")
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("tree mode layout: %+v", problems)
	}
	click(flatModeID)
	if m.treeMode {
		t.Fatal("flat mode was not enabled")
	}
	other := t.TempDir()
	cmd = exec.Command(gitPath, "init", other)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init second repo: %v: %s", err, output)
	}
	m.Open(other)
	wait(func() bool {
		return !m.busy && m.repository != nil && filepath.Clean(m.repository.Root) == filepath.Clean(other)
	})
	if err := os.WriteFile(filepath.Join(dir, "sample.txt"), []byte("another old-repo edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "second.txt"), []byte("new repo edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { return !m.busy && len(m.status.Files) == 1 && m.status.Files[0].Path == "second.txt" })
	m.message.SetText("draft survives failed open")
	m.Open(filepath.Join(other, "missing"))
	wait(func() bool { return !m.busy })
	if m.operationError == "" || filepath.Clean(m.repository.Root) != filepath.Clean(other) || m.message.Text() != "draft survives failed open" {
		t.Fatalf("failed open lost context: repository=%s error=%q draft=%q", m.repository.Root, m.operationError, m.message.Text())
	}
	click(errorDetailsID)
	if !m.showErrorDetails {
		t.Fatal("Git error details were not shown")
	}
}
