package workbench

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/vivaui/viva/components"

	"r-git-workbench/internal/git"
	"r-git-workbench/internal/repo"
)

type Detection struct {
	Installation git.Installation
	Err          error
}

type Session struct {
	mu      sync.Mutex
	pending []Detection
	updates []Update
	writes  sync.Mutex
}

type Update struct {
	Repository   *repo.Repository
	Status       git.Status
	Diff         string
	Comparison   repo.FileDiff
	Aligned      components.CodeDiffResult
	Path         string
	Staged       bool
	Kind         string
	Err          error
	Request      uint64
	Commits      []git.Commit
	Skip         int
	Message      string
	Files        []string
	Refs         []git.Ref
	BranchPlan   repo.BranchDeletePlan
	CheckoutPlan repo.RemoteCheckoutPlan
	PushPlan     repo.ForcePushPlan
	PushPreview  repo.PushPlan
}

func (session *Session) Refs(ctx context.Context, r *repo.Repository, request uint64, notify func()) {
	go func() {
		refs, err := r.Refs(ctx)
		session.deliver(Update{Repository: r, Kind: "refs", Request: request, Refs: refs, Err: err}, notify)
	}()
}

func (session *Session) Log(ctx context.Context, r *repo.Repository, skip int, filter repo.LogFilter, request uint64, notify func()) {
	go func() {
		commits, err := r.LogQuery(ctx, skip, 100, filter)
		session.deliver(Update{Repository: r, Kind: "log", Commits: commits, Skip: skip, Request: request, Err: err}, notify)
	}()
}

func (session *Session) CommitDetails(ctx context.Context, r *repo.Repository, sha string, request uint64, notify func()) {
	go func() {
		message, files, err := r.CommitDetails(ctx, sha)
		session.deliver(Update{Repository: r, Kind: "commit-details", Path: sha, Message: message, Files: files, Request: request, Err: err}, notify)
	}()
}

func (session *Session) CommitFile(ctx context.Context, r *repo.Repository, sha, path string, request uint64, notify func()) {
	go func() {
		comparison, err := r.CommitFile(ctx, sha, path)
		var aligned components.CodeDiffResult
		if err == nil && !comparison.Binary {
			aligned = components.DiffCodeLines(comparison.Left, comparison.Right)
		}
		session.deliver(Update{Repository: r, Kind: "commit-file", Path: path, Message: sha, Comparison: comparison, Aligned: aligned, Request: request, Err: err}, notify)
	}()
}

func (session *Session) deliver(update Update, notify func()) {
	session.mu.Lock()
	session.updates = append(session.updates, update)
	session.mu.Unlock()
	notify()
}

func (session *Session) WatchError(r *repo.Repository, err error, notify func()) {
	session.deliver(Update{Repository: r, Kind: "watch-error", Err: err}, notify)
}

func (session *Session) Changed(r *repo.Repository, notify func()) {
	session.deliver(Update{Repository: r, Kind: "changed"}, notify)
}

func (session *Session) Open(ctx context.Context, gitPath, directory string, request uint64, notify func()) {
	go func() {
		r, status, err := repo.Open(ctx, gitPath, directory)
		session.deliver(Update{Repository: r, Status: status, Kind: "open", Err: err, Request: request}, notify)
	}()
}

func (session *Session) Refresh(ctx context.Context, r *repo.Repository, notify func()) {
	go func() {
		status, err := r.Status(ctx)
		session.deliver(Update{Repository: r, Status: status, Kind: "refresh", Err: err}, notify)
	}()
}

func (session *Session) Diff(ctx context.Context, r *repo.Repository, file git.File, staged bool, request uint64, notify func()) {
	go func() {
		comparison, err := r.Compare(ctx, file, staged)
		var aligned components.CodeDiffResult
		if err == nil && !comparison.Binary {
			aligned = components.DiffCodeLines(comparison.Left, comparison.Right)
		}
		session.deliver(Update{Repository: r, Path: file.Path, Staged: staged, Diff: comparison.Patch, Comparison: comparison, Aligned: aligned, Kind: "diff", Err: err, Request: request}, notify)
	}()
}

func (session *Session) Write(ctx context.Context, r *repo.Repository, kind, path, message string, notify func()) {
	go func() {
		session.writes.Lock()
		defer session.writes.Unlock()
		var err error
		switch kind {
		case "stage":
			err = r.Stage(ctx, path)
		case "unstage":
			err = r.Unstage(ctx, path)
		case "commit":
			err = r.Commit(ctx, message)
		case "create-branch":
			err = r.CreateBranch(ctx, path)
		case "switch-branch":
			err = r.SwitchBranch(ctx, path)
		case "rename-branch":
			parts := strings.SplitN(message, "\x00", 2)
			if len(parts) != 2 {
				err = errors.New("invalid branch rename request")
			} else {
				err = r.RenameBranch(ctx, parts[0], parts[1])
			}
		case "set-upstream":
			parts := strings.SplitN(message, "\x00", 2)
			if len(parts) != 2 {
				err = errors.New("invalid upstream request")
			} else {
				err = r.SetUpstream(ctx, parts[0], parts[1])
			}
		case "checkout-remote":
			parts := strings.SplitN(message, "\x00", 2)
			if len(parts) != 2 {
				err = errors.New("invalid remote checkout request")
			} else {
				_, err = r.PlanCheckoutRemoteBranch(ctx, parts[0], parts[1])
			}
		}
		status, refreshErr := r.Status(ctx)
		if err == nil {
			err = refreshErr
		}
		session.deliver(Update{Repository: r, Status: status, Path: path, Kind: kind, Err: err}, notify)
	}()
}

func (session *Session) PlanRemoteCheckout(ctx context.Context, r *repo.Repository, remote, local string, request uint64, notify func()) {
	go func() {
		plan, err := r.PlanCheckoutRemoteBranch(ctx, remote, local)
		session.deliver(Update{Repository: r, Kind: "remote-checkout-plan", Path: local, CheckoutPlan: plan, Request: request, Err: err}, notify)
	}()
}

func (session *Session) ExecuteRemoteCheckout(ctx context.Context, r *repo.Repository, plan repo.RemoteCheckoutPlan, mode repo.CheckoutMode, notify func()) {
	go func() {
		session.writes.Lock()
		defer session.writes.Unlock()
		err := r.ExecuteCheckoutRemoteBranch(ctx, plan, mode)
		status, refreshErr := r.Status(ctx)
		if err == nil {
			err = refreshErr
		}
		session.deliver(Update{Repository: r, Status: status, Path: plan.Local, Kind: "checkout-remote", Err: err}, notify)
	}()
}

func (session *Session) Remote(ctx context.Context, r *repo.Repository, operation string, policy repo.PullPolicy, notify func()) {
	go func() {
		session.writes.Lock()
		defer session.writes.Unlock()
		err := r.RemoteWithPolicy(ctx, operation, policy, func(line string) {
			session.deliver(Update{Repository: r, Kind: "remote-progress", Message: line}, notify)
		})
		status, refreshErr := r.Status(ctx)
		if err == nil {
			err = refreshErr
		}
		session.deliver(Update{Repository: r, Status: status, Kind: "remote-" + operation, Err: err}, notify)
	}()
}

func (session *Session) PlanForcePush(ctx context.Context, r *repo.Repository, notify func()) {
	go func() {
		plan, err := r.PlanForcePush(ctx)
		session.deliver(Update{Repository: r, Kind: "force-push-plan", PushPlan: plan, Err: err}, notify)
	}()
}

func (session *Session) PlanPush(ctx context.Context, r *repo.Repository, notify func()) {
	go func() {
		plan, err := r.PlanPush(ctx)
		session.deliver(Update{Repository: r, Kind: "push-plan", PushPreview: plan, Err: err}, notify)
	}()
}

func (session *Session) ExecutePush(ctx context.Context, r *repo.Repository, plan repo.PushPlan, notify func()) {
	go func() {
		session.writes.Lock()
		defer session.writes.Unlock()
		err := r.ExecutePush(ctx, plan, func(line string) {
			session.deliver(Update{Repository: r, Kind: "remote-progress", Message: line}, notify)
		})
		status, refreshErr := r.Status(ctx)
		if err == nil {
			err = refreshErr
		}
		session.deliver(Update{Repository: r, Status: status, Kind: "remote-push", Err: err}, notify)
	}()
}

func (session *Session) ForcePush(ctx context.Context, r *repo.Repository, plan repo.ForcePushPlan, notify func()) {
	go func() {
		session.writes.Lock()
		defer session.writes.Unlock()
		err := r.ExecuteForcePush(ctx, plan, func(line string) {
			session.deliver(Update{Repository: r, Kind: "remote-progress", Message: line}, notify)
		})
		status, refreshErr := r.Status(ctx)
		if err == nil {
			err = refreshErr
		}
		session.deliver(Update{Repository: r, Status: status, Kind: "remote-push", Err: err}, notify)
	}()
}

func (session *Session) PlanDeleteBranch(ctx context.Context, r *repo.Repository, name string, request uint64, notify func()) {
	go func() {
		plan, err := r.PlanDeleteBranch(ctx, name)
		session.deliver(Update{Repository: r, Kind: "branch-delete-plan", Path: name, BranchPlan: plan, Request: request, Err: err}, notify)
	}()
}

func (session *Session) DeleteBranch(ctx context.Context, r *repo.Repository, plan repo.BranchDeletePlan, notify func()) {
	go func() {
		session.writes.Lock()
		defer session.writes.Unlock()
		err := r.DeleteBranch(ctx, plan)
		status, refreshErr := r.Status(ctx)
		if err == nil {
			err = refreshErr
		}
		session.deliver(Update{Repository: r, Status: status, Path: plan.Name, Kind: "delete-branch", Err: err}, notify)
	}()
}

func (session *Session) DrainUpdates() []Update {
	session.mu.Lock()
	defer session.mu.Unlock()
	updates := session.updates
	session.updates = nil
	return updates
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
