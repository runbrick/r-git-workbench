//go:build windows

package fswatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchReportsChangesAndStopsAfterCancel(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	changes := make(chan struct{}, 1)
	if err := Watch(ctx, []string{dir}, func() {
		select {
		case changes <- struct{}{}:
		default:
		}
	}, func(err error) { t.Errorf("watch failed: %v", err) }); err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "external.txt"), []byte("change"), 0600); err != nil {
		cancel()
		t.Fatal(err)
	}
	select {
	case <-changes:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("directory change was not reported")
	}
	cancel()
	// Cancellation must release the watched directory so it can be renamed.
	from, to := filepath.Join(dir, "external.txt"), filepath.Join(dir, "renamed.txt")
	if err := os.Rename(from, to); err != nil {
		t.Fatalf("rename after cancel: %v", err)
	}
}
