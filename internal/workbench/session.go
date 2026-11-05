package workbench

import (
	"context"
	"sync"

	"r-git-workbench/internal/git"
)

type Detection struct {
	Installation git.Installation
	Err          error
}

type Session struct {
	mu      sync.Mutex
	pending []Detection
}

func (session *Session) DetectGit(ctx context.Context, path string, notify func()) {
	go func() {
		installation, err := git.Detect(ctx, path)
		session.mu.Lock()
		session.pending = append(session.pending, Detection{Installation: installation, Err: err})
		session.mu.Unlock()
		notify()
	}()
}

func (session *Session) Drain() []Detection {
	session.mu.Lock()
	defer session.mu.Unlock()
	result := session.pending
	session.pending = nil
	return result
}
