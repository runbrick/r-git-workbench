package git

import (
	"context"
	"os/exec"
	"testing"
)

func TestParseVersion(t *testing.T) {
	for _, test := range []struct {
		output string
		write  bool
		valid  bool
	}{
		{"git version 2.53.0.windows.1\n", true, true},
		{"git version 2.34.0\n", true, true},
		{"git version 2.33.9\n", false, true},
		{"git version 3.0.0\n", true, true},
		{"not git\n", false, false},
	} {
		v, err := ParseVersion(test.output)
		if (err == nil) != test.valid {
			t.Fatalf("ParseVersion(%q) error = %v", test.output, err)
		}
		if test.valid && v.Capabilities().CanWrite != test.write {
			t.Fatalf("ParseVersion(%q) CanWrite = %t", test.output, v.Capabilities().CanWrite)
		}
	}
}

func TestDetectSystemGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("system Git is unavailable")
	}
	installation, err := Detect(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if installation.Path == "" || installation.Version.Raw == "" {
		t.Fatalf("incomplete installation: %+v", installation)
	}
}
