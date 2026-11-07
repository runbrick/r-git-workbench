package shell

import (
	"fmt"
	"strings"
	"testing"

	"r-git-workbench/internal/git"
)

func TestGraphKeepsParentsAcrossPagesAndMerge(t *testing.T) {
	commits := []git.Commit{
		{SHA: "merge", Parents: []string{"main", "feature"}},
		{SHA: "main", Parents: []string{"base"}},
		{SHA: "feature", Parents: []string{"base"}},
		{SHA: "base"},
	}
	rows := buildGraph(commits)
	if rows[0].lane != 0 || rows[0].after[0] != "main" || rows[0].after[1] != "feature" || !rows[0].merge[1] {
		t.Fatalf("merge lanes: %+v", rows[0])
	}
	if rows[1].lane != 0 || rows[1].after[0] != "base" || rows[1].after[1] != "feature" {
		t.Fatalf("first-parent continuation: %+v", rows[1])
	}
	if rows[2].lane != 1 || rows[2].after[0] != "base" || rows[2].after[1] != "" {
		t.Fatalf("feature joins base: %+v", rows[2])
	}
	if rows[3].lane != 0 || len(rows[3].after) != 2 || rows[3].after[0] != "" || rows[3].after[1] != "" {
		t.Fatalf("root lane: %+v", rows[3])
	}
}

func TestGraphExpandsForMoreThanSixParallelBranches(t *testing.T) {
	commits := []git.Commit{{SHA: "join"}}
	for i := 0; i < 8; i++ {
		commits[0].Parents = append(commits[0].Parents, fmt.Sprintf("branch-%d", i))
	}
	for i := 0; i < 8; i++ {
		commits = append(commits, git.Commit{SHA: fmt.Sprintf("branch-%d", i), Parents: []string{"base"}})
	}
	commits = append(commits, git.Commit{SHA: "base"})

	rows := buildGraph(commits)
	seen := make(map[int]string)
	for i := 1; i <= 8; i++ {
		lane := rows[i].lane
		if previous, exists := seen[lane]; exists {
			t.Fatalf("independent branches %s and %s share lane %d", previous, commits[i].SHA, lane)
		}
		seen[lane] = commits[i].SHA
	}
	if len(seen) != 8 {
		t.Fatalf("got %d distinct lanes for 8 live branches", len(seen))
	}
	if rows[0].after[6] != "branch-6" || rows[0].after[7] != "branch-7" {
		t.Fatalf("merge parents were lost: %+v", rows[0].after)
	}
}

func TestPaintRefLabelsDistinguishesHeadBranchRemoteAndTag(t *testing.T) {
	sha := "abc123"
	refs := []git.Ref{
		{Name: "main", SHA: sha},
		{Name: "origin/main", SHA: sha, Remote: true},
		{Name: "v1.0", SHA: sha, Tag: true},
	}
	labels := refLabels(sha, refs, sha)
	for _, want := range []string{"HEAD", "branch: main", "remote: origin/main", "tag: v1.0"} {
		if !strings.Contains(labels, want) {
			t.Errorf("missing %q in ref labels %q", want, labels)
		}
	}
}
