//go:build linux

package fswatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchNestedAndNewDirectories(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := make(chan struct{}, 32)
	if err := Watch(ctx, []string{root}, func() {
		select {
		case changes <- struct{}{}:
		default:
		}
	}, func(err error) { t.Errorf("watch failed: %v", err) }); err != nil {
		t.Fatal(err)
	}
	wait := func() {
		t.Helper()
		select {
		case <-changes:
		case <-time.After(5 * time.Second):
			t.Fatal("directory event was not reported")
		}
	}
	if err := os.WriteFile(filepath.Join(nested, "first"), []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	wait()
	newDir := filepath.Join(root, "new")
	if err := os.Mkdir(newDir, 0700); err != nil {
		t.Fatal(err)
	}
	wait()
	for len(changes) > 0 {
		<-changes
	}
	if err := os.WriteFile(filepath.Join(newDir, "second"), []byte("b"), 0600); err != nil {
		t.Fatal(err)
	}
	wait()
	cancel()
}
