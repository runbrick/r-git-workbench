package shell

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"

	"r-git-workbench/internal/git"
	"r-git-workbench/internal/i18n"
	"r-git-workbench/internal/keymap"
	"r-git-workbench/internal/repo"
)

func TestNavigationIsNamedAndSwitchesViews(t *testing.T) {
	m := New(i18n.English)
	var canvas ui.Canvas
	driver, err := app.NewDriver(m.Config(), &canvas)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	for i, action := range keymap.Actions {
		found := false
		for _, hit := range driver.Snapshot().Hits {
			if hit.ID != ids[i] {
				continue
			}
			found = true
			if hit.Label != m.Language.Text(action.Name) || !strings.HasPrefix(tree.Text(hit.ID), "shell/") {
				t.Fatalf("navigation control has no stable identity or accessible name: %+v", hit)
			}
			bounds := hit.Bounds
			if err := driver.Click(bounds.X+bounds.Width/2, bounds.Y+bounds.Height/2); err != nil {
				t.Fatal(err)
			}
			if m.Active != action.ID {
				t.Fatalf("click on %s selected %d", hit.Label, m.Active)
			}
			break
		}
		if !found {
			t.Fatalf("missing navigation %v", action)
		}
	}
	for _, hit := range driver.Snapshot().Hits {
		if hit.Label == "" || tree.Text(hit.ID) == "" {
			t.Fatalf("unnamed control: %+v", hit)
		}
	}
}

func TestLogTableOnlyDeclaresVisibleRows(t *testing.T) {
	m := New(i18n.English)
	m.Active = keymap.ShowLog
	m.repository = new(repo.Repository)
	m.commits = make([]git.Commit, 10000)
	for i := range m.commits {
		m.commits[i] = git.Commit{SHA: fmt.Sprintf("%040x", i+1), Subject: "commit"}
	}
	m.logSource.commits = m.commits
	for _, size := range [][2]int{{1200, 760}, {720, 480}} {
		driver, err := app.NewDriver(m.Config(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := driver.Resize(size[0], size[1]); err != nil {
			t.Fatal(err)
		}
		visible := 0
		for _, hit := range driver.Snapshot().Hits {
			if _, ok := hit.ID.In(logScope); ok {
				visible++
			}
		}
		if visible < 1 || visible > 45 {
			t.Errorf("%v: %d visible rows from 10000 commits", size, visible)
		}
		driver.Close()
	}
}

func BenchmarkNavigationPaint(b *testing.B) {
	m := New(i18n.English)
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		b.Fatal(err)
	}
	defer driver.Close()
	var bounds ui.Rect
	for _, hit := range driver.Snapshot().Hits {
		if hit.ID == ids[0] {
			bounds = hit.Bounds
			break
		}
	}
	if bounds.Width == 0 {
		b.Fatal("Changes navigation is missing")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := driver.Click(bounds.X+bounds.Width/2, bounds.Y+bounds.Height/2); err != nil {
			b.Fatal(err)
		}
	}
}
