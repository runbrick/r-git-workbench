package shell

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/headless"
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
			t.Fatalf("unnamed control %s: %+v", tree.Text(hit.ID), hit)
		}
	}
}

func TestShortcutNavigatesWithoutStealingTextInput(t *testing.T) {
	m := New(i18n.English)
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if handled, err := driver.ShortcutChord(ui.ModifierControl, '4'); err != nil || !handled || m.Active != keymap.ShowStash {
		t.Fatalf("view shortcut: handled=%t active=%d err=%v", handled, m.Active, err)
	}
	if handled, err := driver.ShortcutChord(ui.ModifierControl, 'x'); err != nil || handled {
		t.Fatalf("unknown shortcut was consumed: handled=%t err=%v", handled, err)
	}
}

func TestStashViewAndConflictActionsHaveNamedControls(t *testing.T) {
	m := New(i18n.English)
	m.Active = keymap.ShowStash
	m.repository = new(repo.Repository)
	m.Git.Version = git.Version{Major: 2, Minor: 40}
	m.stashes = []repo.Stash{{Ref: "stash@{0}", SHA: strings.Repeat("a", 40), Subject: "saved"}}
	m.selectedStash = m.stashes[0].SHA
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	for _, id := range []ui.ID{stashSaveID, stashUntrackedID, stashBranchNameID, tree.ID(feature, "stash-apply"), tree.ID(feature, "stash-pop"), tree.ID(feature, "stash-drop")} {
		found := false
		for _, hit := range driver.Snapshot().Hits {
			found = found || hit.ID == id
		}
		if !found {
			t.Errorf("stash control %s missing", tree.Text(id))
		}
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("stash layout: %+v", problems)
	}
	m.Active = keymap.ShowChanges
	m.status.Files = []git.File{{Path: "conflict.txt", Conflict: true}}
	m.selected = 0
	driver2, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver2.Close()
	for _, id := range []ui.ID{tree.ID(feature, "accept-current"), tree.ID(feature, "accept-incoming"), tree.ID(feature, "accept-both")} {
		found := false
		for _, hit := range driver2.Snapshot().Hits {
			found = found || hit.ID == id
		}
		if !found {
			t.Errorf("conflict control %s missing", tree.Text(id))
		}
	}
}

func TestTagViewHasNamedCreateAndDeleteControls(t *testing.T) {
	m := New(i18n.English)
	m.Active = keymap.ShowTags
	m.repository = new(repo.Repository)
	m.Git.Version = git.Version{Major: 2, Minor: 40}
	m.refs = []git.Ref{{Name: "v1.0", FullName: "refs/tags/v1.0", SHA: strings.Repeat("a", 40), Tag: true}}
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	for _, id := range []ui.ID{tagNameID, tagTargetID, tagMessageID, tree.ID(feature, "tag-create"), tree.Sub(feature, tree.Name("tag-delete")).At(0)} {
		found := false
		for _, hit := range driver.Snapshot().Hits {
			found = found || hit.ID == id
		}
		if !found {
			t.Errorf("tag control %s missing", tree.Text(id))
		}
	}
	if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
		t.Fatalf("tag layout: %+v", problems)
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

func TestHistoryToolsCollapseBelowWideLayout(t *testing.T) {
	m := New(i18n.English)
	m.Active = keymap.ShowLog
	m.repository = new(repo.Repository)
	for _, size := range [][2]int{{720, 480}, {1100, 720}, {1400, 900}} {
		driver, err := app.NewDriver(m.Config(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := driver.Resize(size[0], size[1]); err != nil {
			t.Fatal(err)
		}
		if problems := headless.CheckLayout(driver.Snapshot()); len(problems) != 0 {
			t.Errorf("%vx%v layout: %+v", size[0], size[1], problems)
		}
		found := false
		for _, hit := range driver.Snapshot().Hits {
			found = found || hit.ID == tree.ID(feature, "history-tools-toggle")
		}
		if (size[0] < 1000) != found {
			t.Errorf("%vx%v history toggle visible = %t", size[0], size[1], found)
		}
		driver.Close()
	}
}

func TestAbortOperationRequiresConfirmation(t *testing.T) {
	m := New(i18n.English)
	m.Active = keymap.ShowLog
	m.repository = new(repo.Repository)
	m.Git.Version = git.Version{Major: 2, Minor: 40}
	m.status.Operation.Kind = git.OperationMerge
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
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
		t.Fatalf("missing control %s", tree.Text(id))
	}
	click(abortOperationID)
	if m.operationConfirm != "abort" || m.busy {
		t.Fatalf("abort did not wait for confirmation: confirm=%q busy=%t", m.operationConfirm, m.busy)
	}
	click(tree.ID(feature, "operation-cancel"))
	if m.operationConfirm != "" || m.busy {
		t.Fatalf("cancel changed operation state: confirm=%q busy=%t", m.operationConfirm, m.busy)
	}
}

func TestSkipOperationOnlyVisibleForSequencedHistory(t *testing.T) {
	for _, kind := range []git.OperationKind{git.OperationMerge, git.OperationRebase, git.OperationCherryPick} {
		m := New(i18n.English)
		m.Active = keymap.ShowLog
		m.repository = new(repo.Repository)
		m.Git.Version = git.Version{Major: 2, Minor: 40}
		m.status.Operation.Kind = kind
		driver, err := app.NewDriver(m.Config(), nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, hit := range driver.Snapshot().Hits {
			found = found || hit.ID == skipOperationID
		}
		if found != (kind == git.OperationRebase || kind == git.OperationCherryPick) {
			t.Errorf("operation %d skip visibility=%t", kind, found)
		}
		driver.Close()
	}
}

func TestRebaseProgressIsVisible(t *testing.T) {
	m := New(i18n.English)
	m.Active = keymap.ShowLog
	m.repository = new(repo.Repository)
	m.Git.Version = git.Version{Major: 2, Minor: 40}
	m.status.Operation = git.OperationState{Kind: git.OperationRebase, Target: "0123456789abcdef", Current: 2, Remaining: 3}
	driver, err := app.NewDriver(m.Config(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	progress := false
	for _, painted := range driver.Snapshot().Texts {
		progress = progress || painted.Value == "Progress 2/5"
	}
	if !progress {
		t.Fatal("rebase progress label is missing")
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
