package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"

	"r-git-workbench/internal/git"
	"r-git-workbench/internal/i18n"
	"r-git-workbench/internal/repo"
)

func TestLogScrollLoadsNextPage(t *testing.T) {
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
	run("config", "user.name", "Scroll Test")
	run("config", "user.email", "scroll@example.com")
	for i := range 105 {
		run("commit", "--allow-empty", "-m", fmt.Sprintf("entry %03d", i))
	}
	r, status, err := repo.Open(context.Background(), gitPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(i18n.English)
	m.SetInitialGit(git.Installation{Path: gitPath, Version: git.Version{Major: 2, Minor: 40}}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.SetContext(ctx)
	m.SetInitialRepository(r, status, nil)
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	wait := func(want int) {
		t.Helper()
		deadline := time.After(10 * time.Second)
		for m.logLoading || len(m.commits) != want {
			select {
			case <-driver.Wakes():
				if err := driver.Pump(); err != nil {
					t.Fatal(err)
				}
			case <-deadline:
				t.Fatalf("loaded %d commits, want %d: %s", len(m.commits), want, m.operationError)
			}
		}
	}
	wait(100)
	bounds := m.logTableBounds.Rect()
	if err := driver.Scroll(ui.ScrollEvent{X: bounds.X + bounds.Width/2, Y: bounds.Y + bounds.Height/2, DeltaY: 10000}); err != nil {
		t.Fatal(err)
	}
	wait(105)
	if m.commits[100].Subject != "entry 004" || m.commits[104].Subject != "entry 000" || m.graphRows[99].after[0] != m.commits[100].SHA {
		t.Fatalf("pagination/graph boundary: commit[100]=%+v, commit[104]=%+v, graph=%+v", m.commits[100], m.commits[104], m.graphRows[99])
	}
}
