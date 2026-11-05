package shell

import (
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
	if rows[3].lane != 0 || rows[3].after != ([graphLanes]string{}) {
		t.Fatalf("root lane: %+v", rows[3])
	}
}
