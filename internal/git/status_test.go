package git

import (
	"strings"
	"testing"
)

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
