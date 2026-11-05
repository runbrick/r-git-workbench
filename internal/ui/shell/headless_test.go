package shell

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vivaui/viva/headless"

	"r-git-workbench/internal/i18n"
)

func TestHeadlessNavigationAndLayout(t *testing.T) {
	session, err := headless.Open(New(i18n.English).Config(), nil, headless.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	var output bytes.Buffer
	if err := headless.Serve(session, strings.NewReader("hits\nlayout\nquit\n"), &output); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"changes", "log", "branches", "stash", "tags", "repositories"} {
		if !strings.Contains(output.String(), "id=shell/"+name+" ") {
			t.Fatalf("missing named navigation control %q:\n%s", name, &output)
		}
	}
	if !strings.Contains(output.String(), "ok 0 problems") {
		t.Fatalf("headless layout has problems:\n%s", &output)
	}
	for _, hit := range session.Snapshot().Hits {
		if hit.Label == "" {
			t.Fatalf("headless control has no accessible name: %+v", hit)
		}
	}
}

func TestNarrowWindowLayout(t *testing.T) {
	session, err := headless.Open(New(i18n.Chinese).Config(), nil, headless.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Driver().Resize(720, 480); err != nil {
		t.Fatal(err)
	}
	if problems := headless.CheckLayout(session.Snapshot()); len(problems) != 0 {
		t.Fatalf("narrow window has layout problems: %+v", problems)
	}
}
