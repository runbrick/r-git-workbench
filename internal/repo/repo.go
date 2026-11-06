package repo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"r-git-workbench/internal/fswatch"
	"r-git-workbench/internal/git"
	"r-git-workbench/internal/gitexec"
)

type Repository struct {
	Root   string
	runner gitexec.Runner
	write  sync.Mutex
}

func (r *Repository) Refs(ctx context.Context) ([]git.Ref, error) {
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{
		"for-each-ref", "--format=%(refname)%00%(objectname)%00%(upstream)%00%(upstream:trackshort)%00", "refs/heads", "refs/remotes",
	}, OutputLimit: 4 << 20})
	if err != nil {
		return nil, fmt.Errorf("list branches: %w: %s", err, result.Stderr)
	}
	return git.ParseRefs(result.Stdout)
}

func (r *Repository) Log(ctx context.Context, skip, limit int) ([]git.Commit, error) {
	return r.LogQuery(ctx, skip, limit, LogFilter{})
}

func (r *Repository) LogFiltered(ctx context.Context, skip, limit int, message string) ([]git.Commit, error) {
	return r.LogQuery(ctx, skip, limit, LogFilter{Message: message})
}

type LogFilter struct {
	Message string
	Author  string
	Path    string
	Ref     string
	Since   string
	Until   string
}

func (r *Repository) LogQuery(ctx context.Context, skip, limit int, filter LogFilter) ([]git.Commit, error) {
	if skip < 0 || limit < 1 || limit > 200 {
		return nil, errors.New("invalid log page")
	}
	args := []string{"log", "--topo-order", "-z", "--format=%H%x00%P%x00%an%x00%aI%x00%s%x00", fmt.Sprintf("--skip=%d", skip), fmt.Sprintf("-n%d", limit)}
	if filter.Ref == "" {
		args = append(args, "--all")
	} else {
		if !strings.HasPrefix(filter.Ref, "refs/heads/") && !strings.HasPrefix(filter.Ref, "refs/remotes/") || strings.ContainsAny(filter.Ref, "\x00\r\n") {
			return nil, errors.New("invalid branch reference")
		}
		args = append(args, filter.Ref)
	}
	if filter.Message != "" {
		args = append(args, "--fixed-strings", "--regexp-ignore-case", "--grep="+filter.Message)
	}
	if filter.Author != "" {
		args = append(args, "--fixed-strings", "--regexp-ignore-case", "--author="+filter.Author)
	}
	if filter.Since != "" {
		start, err := time.Parse("2006-01-02", filter.Since)
		if err != nil {
			return nil, errors.New("invalid log start date; expected YYYY-MM-DD")
		}
		args = append(args, fmt.Sprintf("--since=@%d", start.Unix()))
	}
	if filter.Until != "" {
		end, err := time.Parse("2006-01-02", filter.Until)
		if err != nil {
			return nil, errors.New("invalid log end date; expected YYYY-MM-DD")
		}
		args = append(args, fmt.Sprintf("--before=@%d", end.AddDate(0, 0, 1).Unix()))
	}
	if filter.Since != "" && filter.Until != "" && filter.Since > filter.Until {
		return nil, errors.New("log start date must not be after end date")
	}
	if filter.Path != "" {
		if err := r.checkPath(filter.Path); err != nil {
			return nil, err
		}
		args = append(args, "--", filter.Path)
	}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, OutputLimit: 4 << 20})
	if err != nil {
		// Git exits successfully with no commits only when the ref set is nonempty.
		if strings.Contains(string(result.Stderr), "does not have any commits yet") || strings.Contains(string(result.Stderr), "your current branch") {
			return nil, nil
		}
		return nil, fmt.Errorf("log: %w: %s", err, result.Stderr)
	}
	return git.ParseLog(result.Stdout)
}

func (r *Repository) CommitDetails(ctx context.Context, sha string) (string, []string, error) {
	if err := checkSHA(sha); err != nil {
		return "", nil, err
	}
	parents, err := r.commitParents(ctx, sha)
	if err != nil {
		return "", nil, err
	}
	message, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"show", "-s", "--format=%B", sha}, OutputLimit: 1 << 20})
	if err != nil {
		return "", nil, fmt.Errorf("commit message: %w: %s", err, message.Stderr)
	}
	args := []string{"diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "-z"}
	if len(parents) > 0 {
		args = append(args, parents[0])
	}
	args = append(args, sha)
	files, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, OutputLimit: 4 << 20})
	if err != nil {
		return "", nil, fmt.Errorf("commit files: %w: %s", err, files.Stderr)
	}
	var paths []string
	for _, entry := range bytes.Split(files.Stdout, []byte{0}) {
		if len(entry) > 0 {
			paths = append(paths, string(entry))
		}
	}
	return string(message.Stdout), paths, nil
}

func checkSHA(sha string) error {
	if len(sha) != 40 && len(sha) != 64 {
		return errors.New("invalid commit SHA")
	}
	for _, char := range sha {
		if char < '0' || char > '9' && char < 'a' || char > 'f' {
			return errors.New("invalid commit SHA")
		}
	}
	return nil
}

func (r *Repository) commitParents(ctx context.Context, sha string) ([]string, error) {
	parents, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-list", "--parents", "-n1", sha}})
	if err != nil {
		return nil, fmt.Errorf("commit parents: %w: %s", err, parents.Stderr)
	}
	fields := strings.Fields(string(parents.Stdout))
	if len(fields) == 0 || fields[0] != sha {
		return nil, errors.New("commit not found")
	}
	return fields[1:], nil
}

func (r *Repository) CommitFile(ctx context.Context, sha, path string) (FileDiff, error) {
	if err := r.checkPath(path); err != nil {
		return FileDiff{}, err
	}
	if err := checkSHA(sha); err != nil {
		return FileDiff{}, err
	}
	parents, err := r.commitParents(ctx, sha)
	if err != nil {
		return FileDiff{}, err
	}
	var args []string
	if len(parents) == 0 {
		args = []string{"show", "--format=", "--no-color", "--no-ext-diff", "--no-textconv", "--binary", sha, "--", path}
	} else {
		args = []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv", "--binary", parents[0], sha, "--", path}
	}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, OutputLimit: 8 << 20})
	if err != nil {
		return FileDiff{}, fmt.Errorf("commit file: %w: %s", err, result.Stderr)
	}
	comparison := FileDiff{Patch: string(result.Stdout)}
	comparison.Right, err = r.commitObject(ctx, sha, path)
	if err != nil {
		if errors.Is(err, errTooLarge) {
			comparison.Binary = true
			return comparison, nil
		}
		return FileDiff{}, err
	}
	if len(parents) > 0 {
		comparison.Left, err = r.commitObject(ctx, parents[0], path)
		if err != nil {
			if errors.Is(err, errTooLarge) {
				comparison.Binary = true
				return comparison, nil
			}
			return FileDiff{}, err
		}
	}
	comparison.Binary = strings.IndexByte(comparison.Left, 0) >= 0 || strings.IndexByte(comparison.Right, 0) >= 0
	return comparison, nil
}

func (r *Repository) commitObject(ctx context.Context, sha, path string) (string, error) {
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"ls-tree", "-z", sha, "--", path}, OutputLimit: 1 << 20})
	if err != nil {
		return "", fmt.Errorf("commit tree: %w: %s", err, result.Stderr)
	}
	for _, entry := range bytes.Split(result.Stdout, []byte{0}) {
		parts := bytes.SplitN(entry, []byte{'\t'}, 2)
		if len(parts) == 2 && string(parts[1]) == path {
			return r.object(ctx, sha+":"+path)
		}
	}
	return "", nil
}

type FileDiff struct {
	Patch  string
	Left   string
	Right  string
	Binary bool
}

var errTooLarge = errors.New("file exceeds comparison size limit")

func Open(ctx context.Context, gitPath, directory string) (*Repository, git.Status, error) {
	runner := gitexec.Runner{Path: gitPath}
	status, err := git.ReadStatus(ctx, runner, directory)
	if err != nil {
		return nil, git.Status{}, err
	}
	return &Repository{Root: status.Root, runner: runner}, status, nil
}

func (r *Repository) Status(ctx context.Context) (git.Status, error) {
	return git.ReadStatus(ctx, r.runner, r.Root)
}

type PullPolicy uint8

const (
	PullMerge PullPolicy = iota
	PullRebase
	PullFFOnly
)

func (r *Repository) Remote(ctx context.Context, operation string, progress func(string)) error {
	return r.RemoteWithPolicy(ctx, operation, PullMerge, progress)
}

func (r *Repository) RemoteWithPolicy(ctx context.Context, operation string, policy PullPolicy, progress func(string)) error {
	var args []string
	switch operation {
	case "fetch":
		args = []string{"fetch", "--progress", "--prune"}
	case "pull":
		args = []string{"pull", "--progress"}
		switch policy {
		case PullMerge:
			args = append(args, "--no-rebase")
		case PullRebase:
			args = append(args, "--rebase")
		case PullFFOnly:
			args = append(args, "--ff-only")
		default:
			return fmt.Errorf("unknown pull policy %d", policy)
		}
	case "push":
		args = []string{"push", "--progress"}
	default:
		return fmt.Errorf("unknown remote operation %q", operation)
	}
	r.write.Lock()
	defer r.write.Unlock()
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, Write: true, OnProgress: progress})
	if err != nil {
		return fmt.Errorf("%s: %w: %s", operation, err, result.Stderr)
	}
	return nil
}

type ForcePushPlan struct {
	Remote string
	Branch string
	Local  string
	Lease  string
}

type PushPlan struct {
	Remote   string
	Branch   string
	Local    string
	Upstream string
	Commits  []git.Commit
}

func (r *Repository) PlanPush(ctx context.Context) (PushPlan, error) {
	status, err := r.Status(ctx)
	if err != nil {
		return PushPlan{}, err
	}
	if status.Branch == "" || status.Branch == "(detached)" {
		return PushPlan{}, errors.New("push requires a local branch")
	}
	upstream, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"for-each-ref", "--format=%(upstream)", "refs/heads/" + status.Branch}})
	if err != nil {
		return PushPlan{}, fmt.Errorf("read push upstream: %w: %s", err, upstream.Stderr)
	}
	upstreamRef := strings.TrimSpace(string(upstream.Stdout))
	remote, branch, ok := strings.Cut(strings.TrimPrefix(upstreamRef, "refs/remotes/"), "/")
	if !strings.HasPrefix(upstreamRef, "refs/remotes/") || !ok || remote == "" || branch == "" {
		return PushPlan{}, errors.New("current branch has no usable upstream")
	}
	local, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", "HEAD"}})
	if err != nil {
		return PushPlan{}, fmt.Errorf("resolve local push target: %w: %s", err, local.Stderr)
	}
	remoteSHA, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", upstreamRef}})
	if err != nil {
		return PushPlan{}, fmt.Errorf("resolve remote push target: %w: %s", err, remoteSHA.Stderr)
	}
	commits, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"log", "--reverse", "--topo-order", "-z", "--format=%H%x00%P%x00%an%x00%aI%x00%s%x00", upstreamRef + "..HEAD"}, OutputLimit: 4 << 20})
	if err != nil {
		return PushPlan{}, fmt.Errorf("list commits to push: %w: %s", err, commits.Stderr)
	}
	preview, err := git.ParseLog(commits.Stdout)
	if err != nil {
		return PushPlan{}, err
	}
	return PushPlan{Remote: remote, Branch: branch, Local: strings.TrimSpace(string(local.Stdout)), Upstream: strings.TrimSpace(string(remoteSHA.Stdout)), Commits: preview}, nil
}

func (r *Repository) ExecutePush(ctx context.Context, plan PushPlan, progress func(string)) error {
	if plan.Remote == "" || plan.Branch == "" || plan.Local == "" || plan.Upstream == "" {
		return errors.New("invalid push plan")
	}
	r.write.Lock()
	defer r.write.Unlock()
	current, err := r.PlanPush(ctx)
	if err != nil {
		return err
	}
	if current.Remote != plan.Remote || current.Branch != plan.Branch || current.Local != plan.Local || current.Upstream != plan.Upstream || !sameCommits(current.Commits, plan.Commits) {
		return errors.New("push targets changed after preflight; review a new plan")
	}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"push", "--progress", plan.Remote, "HEAD:refs/heads/" + plan.Branch}, Write: true, OnProgress: progress})
	if err != nil {
		return fmt.Errorf("push: %w: %s", err, result.Stderr)
	}
	return nil
}

func sameCommits(left, right []git.Commit) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].SHA != right[i].SHA || left[i].Author != right[i].Author || left[i].Date != right[i].Date || left[i].Subject != right[i].Subject || !slices.Equal(left[i].Parents, right[i].Parents) {
			return false
		}
	}
	return true
}

func (r *Repository) PlanForcePush(ctx context.Context) (ForcePushPlan, error) {
	status, err := r.Status(ctx)
	if err != nil {
		return ForcePushPlan{}, err
	}
	if status.Branch == "" || status.Branch == "(detached)" {
		return ForcePushPlan{}, errors.New("force push requires a local branch")
	}
	ref, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"for-each-ref", "--format=%(upstream)", "refs/heads/" + status.Branch}})
	if err != nil {
		return ForcePushPlan{}, fmt.Errorf("read push upstream: %w: %s", err, ref.Stderr)
	}
	upstream := strings.TrimSpace(string(ref.Stdout))
	if !strings.HasPrefix(upstream, "refs/remotes/") {
		return ForcePushPlan{}, errors.New("current branch has no upstream")
	}
	remoteBranch := strings.TrimPrefix(upstream, "refs/remotes/")
	remote, branch, ok := strings.Cut(remoteBranch, "/")
	if !ok || remote == "" || branch == "" {
		return ForcePushPlan{}, errors.New("cannot determine push remote and branch")
	}
	local, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", "HEAD"}})
	if err != nil {
		return ForcePushPlan{}, fmt.Errorf("resolve local push target: %w: %s", err, local.Stderr)
	}
	lease, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", upstream}})
	if err != nil {
		return ForcePushPlan{}, fmt.Errorf("resolve remote push target: %w: %s", err, lease.Stderr)
	}
	return ForcePushPlan{Remote: remote, Branch: branch, Local: strings.TrimSpace(string(local.Stdout)), Lease: strings.TrimSpace(string(lease.Stdout))}, nil
}

func (r *Repository) ExecuteForcePush(ctx context.Context, plan ForcePushPlan, progress func(string)) error {
	if plan.Remote == "" || plan.Branch == "" || plan.Local == "" || plan.Lease == "" {
		return errors.New("invalid force push plan")
	}
	r.write.Lock()
	defer r.write.Unlock()
	current, err := r.PlanForcePush(ctx)
	if err != nil {
		return err
	}
	if current != plan {
		return errors.New("push targets changed after preflight; review a new plan")
	}
	args := []string{"push", "--progress", "--force-with-lease=refs/heads/" + plan.Branch + ":" + plan.Lease, plan.Remote, "HEAD:refs/heads/" + plan.Branch}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, Write: true, OnProgress: progress})
	if err != nil {
		return fmt.Errorf("force push: %w: %s", err, result.Stderr)
	}
	return nil
}

// Watch waits for OS filesystem events. A timer exists only after a change and
// merges bursts from Git and the working tree into a single refresh request.
func (r *Repository) Watch(ctx context.Context, notify func(), report func(error)) error {
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--absolute-git-dir"}})
	if err != nil {
		return fmt.Errorf("locate Git directory: %w: %s", err, result.Stderr)
	}
	dir := strings.TrimSuffix(strings.TrimSuffix(string(result.Stdout), "\n"), "\r")
	var mu sync.Mutex
	var timer *time.Timer
	changed := func() {
		mu.Lock()
		defer mu.Unlock()
		if ctx.Err() != nil {
			return
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(300*time.Millisecond, func() {
			if ctx.Err() == nil {
				notify()
			}
		})
	}
	if err := fswatch.Watch(ctx, []string{r.Root, dir}, changed, report); err != nil {
		return err
	}
	// Close the gap between the initial status read and watch registration.
	if ctx.Err() == nil {
		notify()
	}
	go func() {
		<-ctx.Done()
		mu.Lock()
		if timer != nil {
			timer.Stop()
		}
		mu.Unlock()
	}()
	return nil
}

func (r *Repository) Diff(ctx context.Context, path string, staged bool) (string, error) {
	if err := r.checkPath(path); err != nil {
		return "", err
	}
	args := []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv", "--binary"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, OutputLimit: 8 << 20})
	if err != nil {
		return "", fmt.Errorf("diff: %w: %s", err, result.Stderr)
	}
	if !staged && len(result.Stdout) == 0 {
		// Git's regular diff omits untracked files.
		status, statusErr := r.Status(ctx)
		if statusErr != nil {
			return "", statusErr
		}
		for _, file := range status.Files {
			if file.Path != path || !file.Untracked {
				continue
			}
			filename := filepath.Join(r.Root, filepath.FromSlash(path))
			info, statErr := os.Lstat(filename)
			if statErr != nil {
				return "", statErr
			}
			if !info.Mode().IsRegular() {
				return "", nil
			}
			if info.Size() > 8<<20 {
				return "", errTooLarge
			}
			patch, diffErr := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"diff", "--no-index", "--no-color", "--no-ext-diff", "--binary", "--", os.DevNull, filename}, OutputLimit: 8 << 20})
			if diffErr != nil && (patch.ExitCode != 1 || !strings.Contains(diffErr.Error(), "exit status 1")) {
				return "", fmt.Errorf("untracked diff: %w: %s", diffErr, patch.Stderr)
			}
			return string(patch.Stdout), nil
		}
	}
	return string(result.Stdout), nil
}

// Compare reads the actual two versions represented by a Changes row. The
// original path matters for a staged rename, whose old side is in HEAD.
func (r *Repository) Compare(ctx context.Context, file git.File, staged bool) (FileDiff, error) {
	patch, err := r.Diff(ctx, file.Path, staged)
	if err != nil {
		return FileDiff{}, err
	}
	comparison := FileDiff{Patch: patch}
	if staged {
		if file.Index != 'A' && file.Index != '?' {
			old := file.Path
			if file.Original != "" {
				old = file.Original
			}
			comparison.Left, err = r.object(ctx, "HEAD:"+old)
			if err != nil {
				if errors.Is(err, errTooLarge) {
					comparison.Binary = true
					return comparison, nil
				}
				return FileDiff{}, err
			}
		}
		if file.Index != 'D' {
			comparison.Right, err = r.object(ctx, ":"+file.Path)
			if err != nil {
				if errors.Is(err, errTooLarge) {
					comparison.Binary = true
					return comparison, nil
				}
				return FileDiff{}, err
			}
		}
	} else {
		if !file.Untracked && file.Index != 'D' {
			comparison.Left, err = r.object(ctx, ":"+file.Path)
			if err != nil {
				if errors.Is(err, errTooLarge) {
					comparison.Binary = true
					return comparison, nil
				}
				return FileDiff{}, err
			}
		}
		if file.Worktree != 'D' {
			path := filepath.Join(r.Root, filepath.FromSlash(file.Path))
			info, statErr := os.Lstat(path)
			if statErr != nil {
				return FileDiff{}, statErr
			}
			if !info.Mode().IsRegular() {
				comparison.Binary = true
				return comparison, nil
			}
			if info.Size() > 512<<10 {
				comparison.Binary = true
				return comparison, nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return FileDiff{}, readErr
			}
			comparison.Right = string(data)
		}
	}
	comparison.Binary = len(comparison.Left) > 512<<10 || len(comparison.Right) > 512<<10 || strings.IndexByte(comparison.Left, 0) >= 0 || strings.IndexByte(comparison.Right, 0) >= 0
	return comparison, nil
}

func (r *Repository) object(ctx context.Context, revision string) (string, error) {
	size, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"cat-file", "-s", revision}})
	if err != nil {
		return "", fmt.Errorf("size of %s: %w: %s", revision, err, size.Stderr)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(size.Stdout)), 10, 64)
	if err != nil {
		return "", fmt.Errorf("size of %s: %w", revision, err)
	}
	if n > 512<<10 {
		return "", errTooLarge
	}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"show", revision}, OutputLimit: 512 << 10})
	if err != nil {
		return "", fmt.Errorf("read %s: %w: %s", revision, err, result.Stderr)
	}
	return string(result.Stdout), nil
}

func (r *Repository) checkPath(path string) error {
	clean := filepath.Clean(filepath.FromSlash(path))
	if path == "" || filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.ContainsRune(path, 0) {
		return errors.New("invalid repository path")
	}
	return nil
}

func (r *Repository) Stage(ctx context.Context, path string) error {
	if err := r.checkPath(path); err != nil {
		return err
	}
	r.write.Lock()
	defer r.write.Unlock()
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"add", "--", path}, Write: true})
	if err != nil {
		return fmt.Errorf("stage: %w: %s", err, result.Stderr)
	}
	return nil
}

func (r *Repository) CreateBranch(ctx context.Context, name string) error {
	if err := r.checkBranchName(ctx, name); err != nil {
		return err
	}
	r.write.Lock()
	defer r.write.Unlock()
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"branch", name}, Write: true})
	if err != nil {
		return fmt.Errorf("create branch: %w: %s", err, result.Stderr)
	}
	return nil
}

func (r *Repository) RenameBranch(ctx context.Context, oldName, newName string) error {
	if err := r.checkBranchName(ctx, oldName); err != nil {
		return err
	}
	if err := r.checkBranchName(ctx, newName); err != nil {
		return err
	}
	r.write.Lock()
	defer r.write.Unlock()
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"branch", "-m", oldName, newName}, Write: true})
	if err != nil {
		return fmt.Errorf("rename branch: %w: %s", err, result.Stderr)
	}
	return nil
}

type BranchDeletePlan struct {
	Name       string
	Head       string
	Target     string
	StatusHash string
}

func (r *Repository) PlanDeleteBranch(ctx context.Context, name string) (BranchDeletePlan, error) {
	if err := r.checkBranchName(ctx, name); err != nil {
		return BranchDeletePlan{}, err
	}
	if name == "" {
		return BranchDeletePlan{}, errors.New("invalid branch name")
	}
	head, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", "HEAD"}})
	if err != nil {
		return BranchDeletePlan{}, fmt.Errorf("branch delete preflight head: %w: %s", err, head.Stderr)
	}
	target, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", "refs/heads/" + name}})
	if err != nil {
		return BranchDeletePlan{}, fmt.Errorf("branch not found: %s: %w: %s", name, err, target.Stderr)
	}
	status, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all"}, OutputLimit: 16 << 20})
	if err != nil {
		return BranchDeletePlan{}, fmt.Errorf("branch delete preflight status: %w: %s", err, status.Stderr)
	}
	for _, record := range bytes.Split(status.Stdout, []byte{0}) {
		if strings.Trim(string(record), "\r\n") == "# branch.head "+name {
			return BranchDeletePlan{}, errors.New("cannot delete the current branch")
		}
	}
	return BranchDeletePlan{Name: name, Head: strings.TrimSpace(string(head.Stdout)), Target: strings.TrimSpace(string(target.Stdout)), StatusHash: fmt.Sprintf("%x", sha256.Sum256(status.Stdout))}, nil
}

func (r *Repository) DeleteBranch(ctx context.Context, plan BranchDeletePlan) error {
	if plan.Name == "" || plan.Head == "" || plan.Target == "" || plan.StatusHash == "" {
		return errors.New("invalid branch delete plan")
	}
	if err := r.checkBranchName(ctx, plan.Name); err != nil {
		return err
	}
	r.write.Lock()
	defer r.write.Unlock()
	current, err := r.PlanDeleteBranch(ctx, plan.Name)
	if err != nil {
		return err
	}
	if current != plan {
		return errors.New("repository changed after branch delete preflight; review a new plan")
	}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"branch", "-d", "--", plan.Name}, Write: true})
	if err != nil {
		return fmt.Errorf("delete branch: %w: %s", err, result.Stderr)
	}
	return nil
}

func (r *Repository) SetUpstream(ctx context.Context, name, upstream string) error {
	if err := r.checkBranchName(ctx, name); err != nil {
		return err
	}
	args := []string{"branch", "--set-upstream-to=" + upstream, name}
	if upstream == "" {
		args = []string{"branch", "--unset-upstream", name}
	} else if !strings.HasPrefix(upstream, "refs/remotes/") || strings.ContainsAny(upstream, "\x00\r\n") {
		return errors.New("invalid upstream reference")
	}
	r.write.Lock()
	defer r.write.Unlock()
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, Write: true})
	if err != nil {
		return fmt.Errorf("set branch upstream: %w: %s", err, result.Stderr)
	}
	return nil
}

func (r *Repository) CheckoutRemoteBranch(ctx context.Context, remoteName, localName string) error {
	plan, err := r.PlanCheckoutRemoteBranch(ctx, remoteName, localName)
	if err != nil {
		return err
	}
	return r.ExecuteCheckoutRemoteBranch(ctx, plan, CheckoutSafe)
}

type CheckoutMode uint8

const (
	CheckoutSafe CheckoutMode = iota
	CheckoutStash
	CheckoutForce
)

type RemoteCheckoutPlan struct {
	Remote     string
	Local      string
	Head       string
	Target     string
	StatusHash string
	DirtyFiles []string
}

func (r *Repository) PlanCheckoutRemoteBranch(ctx context.Context, remoteName, localName string) (RemoteCheckoutPlan, error) {
	if err := r.checkBranchName(ctx, localName); err != nil {
		return RemoteCheckoutPlan{}, err
	}
	if remoteName == "" || strings.ContainsAny(remoteName, "\x00\r\n") {
		return RemoteCheckoutPlan{}, errors.New("invalid remote branch")
	}
	if !strings.HasPrefix(remoteName, "refs/remotes/") {
		return RemoteCheckoutPlan{}, errors.New("invalid remote branch")
	}
	remote, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"show-ref", "--verify", "--quiet", remoteName}})
	if err != nil {
		return RemoteCheckoutPlan{}, fmt.Errorf("remote branch not found: %s: %w: %s", remoteName, err, remote.Stderr)
	}
	target, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", remoteName}})
	if err != nil {
		return RemoteCheckoutPlan{}, fmt.Errorf("resolve remote branch: %w: %s", err, target.Stderr)
	}
	if _, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"show-ref", "--verify", "--quiet", "refs/heads/" + localName}}); err == nil {
		return RemoteCheckoutPlan{}, fmt.Errorf("local branch already exists: %s", localName)
	}
	head, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rev-parse", "--verify", "HEAD"}})
	if err != nil {
		return RemoteCheckoutPlan{}, fmt.Errorf("checkout preflight head: %w: %s", err, head.Stderr)
	}
	status, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all"}, OutputLimit: 16 << 20})
	if err != nil {
		return RemoteCheckoutPlan{}, fmt.Errorf("checkout preflight status: %w: %s", err, status.Stderr)
	}
	parsed, err := git.ParseStatus(status.Stdout)
	if err != nil {
		return RemoteCheckoutPlan{}, err
	}
	dirty := make([]string, 0, len(parsed.Files))
	for _, file := range parsed.Files {
		dirty = append(dirty, file.Path)
	}
	slices.Sort(dirty)
	return RemoteCheckoutPlan{
		Remote: remoteName, Local: localName, Head: strings.TrimSpace(string(head.Stdout)),
		Target: strings.TrimSpace(string(target.Stdout)), StatusHash: fmt.Sprintf("%x", sha256.Sum256(status.Stdout)),
		DirtyFiles: dirty,
	}, nil
}

func (r *Repository) ExecuteCheckoutRemoteBranch(ctx context.Context, plan RemoteCheckoutPlan, mode CheckoutMode) error {
	if plan.Remote == "" || plan.Local == "" || plan.Head == "" || plan.Target == "" || plan.StatusHash == "" || mode > CheckoutForce {
		return errors.New("invalid remote checkout plan")
	}
	r.write.Lock()
	defer r.write.Unlock()
	current, err := r.PlanCheckoutRemoteBranch(ctx, plan.Remote, plan.Local)
	if err != nil {
		return err
	}
	if current.Remote != plan.Remote || current.Local != plan.Local || current.Head != plan.Head || current.Target != plan.Target || current.StatusHash != plan.StatusHash {
		return errors.New("repository changed after checkout preflight; review a new plan")
	}
	if len(plan.DirtyFiles) > 0 && mode == CheckoutSafe {
		return errors.New("working tree has changes; choose stash, force checkout, or cancel")
	}
	if mode == CheckoutStash {
		result, stashErr := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"stash", "push", "--include-untracked", "-m", "r-git-workbench: before remote branch checkout"}, Write: true})
		if stashErr != nil {
			return fmt.Errorf("stash before checkout: %w: %s", stashErr, result.Stderr)
		}
	}
	args := []string{"switch", "--track", "-c", plan.Local, plan.Remote}
	if mode == CheckoutForce {
		args = []string{"switch", "--discard-changes", "--track", "-c", plan.Local, plan.Remote}
	}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: args, Write: true})
	if err != nil {
		return fmt.Errorf("checkout remote branch: %w: %s", err, result.Stderr)
	}
	return nil
}

func (r *Repository) checkBranchName(ctx context.Context, name string) error {
	if name == "" || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("invalid branch name")
	}
	valid, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"check-ref-format", "--branch", name}})
	if err != nil {
		return fmt.Errorf("branch name: %w: %s", err, valid.Stderr)
	}
	return nil
}

func (r *Repository) SwitchBranch(ctx context.Context, name string) error {
	if err := r.checkBranchName(ctx, name); err != nil {
		return err
	}
	r.write.Lock()
	defer r.write.Unlock()
	// Resolve the local ref explicitly so a remote branch cannot be implicitly created.
	ref, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"show-ref", "--verify", "--quiet", "refs/heads/" + name}})
	if err != nil {
		return fmt.Errorf("local branch not found: %s: %w: %s", name, err, ref.Stderr)
	}
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"switch", "--", name}, Write: true})
	if err != nil {
		return fmt.Errorf("switch branch: %w: %s", err, result.Stderr)
	}
	return nil
}

func (r *Repository) Unstage(ctx context.Context, path string) error {
	if err := r.checkPath(path); err != nil {
		return err
	}
	r.write.Lock()
	defer r.write.Unlock()
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"restore", "--staged", "--", path}, Write: true})
	if err != nil {
		// An unborn branch has no HEAD to restore from.
		if strings.Contains(string(result.Stderr), "could not resolve") && strings.Contains(string(result.Stderr), "HEAD") || strings.Contains(string(result.Stderr), "invalid reference: HEAD") {
			result, err = r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"rm", "--cached", "--", path}, Write: true})
		}
	}
	if err != nil {
		return fmt.Errorf("unstage: %w: %s", err, result.Stderr)
	}
	return nil
}

func (r *Repository) Commit(ctx context.Context, message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("commit message is empty")
	}
	r.write.Lock()
	defer r.write.Unlock()
	result, err := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"commit", "-F", "-"}, Stdin: strings.NewReader(message), Write: true})
	if err != nil {
		return fmt.Errorf("commit: %w: %s", err, result.Stderr)
	}
	return nil
}
