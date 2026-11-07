package git

import (
	"strings"
	"testing"
)

func TestParseLogIncludesAuthorAndCommitter(t *testing.T) {
	commit, err := ParseLog([]byte(strings.Join([]string{
		"0123456789012345678901234567890123456789", "parent", "Code Author", "History Rewriter",
		"2026-11-06T12:00:00+08:00", "amended commit", "",
	}, "\x00")))
	if err != nil || len(commit) != 1 {
		t.Fatalf("parse log: %+v, %v", commit, err)
	}
	if commit[0].Author != "Code Author" || commit[0].Committer != "History Rewriter" {
		t.Fatalf("author/committer lost: %+v", commit[0])
	}
}
