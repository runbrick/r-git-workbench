package git

import (
	"strings"
	"testing"
)

func TestParseRefsKeepsLocalRemoteAndUpstream(t *testing.T) {
	sha := strings.Repeat("a", 40)
	data := []byte("refs/heads/feature/test\x00" + sha + "\x00refs/remotes/origin/feature/test\x00\n" +
		"refs/remotes/origin/feature/test\x00" + sha + "\x00\x00\n")
	refs, err := ParseRefs(data)
	if err != nil || len(refs) != 2 {
		t.Fatalf("parse refs: %+v, %v", refs, err)
	}
	if refs[0].Name != "feature/test" || refs[0].Remote || refs[0].Upstream != "refs/remotes/origin/feature/test" || refs[0].SHA != sha {
		t.Fatalf("local ref: %+v", refs[0])
	}
	if refs[1].Name != "origin/feature/test" || !refs[1].Remote || refs[1].Upstream != "" {
		t.Fatalf("remote ref: %+v", refs[1])
	}
	if _, err := ParseRefs([]byte("refs/heads/incomplete\x00")); err == nil {
		t.Fatal("accepted a truncated record")
	}
}
