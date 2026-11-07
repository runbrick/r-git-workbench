package shell

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"

	"r-git-workbench/internal/git"
	"r-git-workbench/internal/i18n"
	"r-git-workbench/internal/keymap"
	"r-git-workbench/internal/repo"
	"r-git-workbench/internal/settings"
	"r-git-workbench/internal/workbench"
)

var (
	tree    = ui.Tree{}
	feature = tree.Feature("shell")
	ids     = [...]ui.ID{
		tree.ID(feature, "changes"), tree.ID(feature, "log"),
		tree.ID(feature, "branches"), tree.ID(feature, "stash"),
		tree.ID(feature, "tags"), tree.ID(feature, "repositories"),
	}
	openID              = tree.ID(feature, "open")
	branchID            = tree.ID(feature, "branch")
	branchNameID        = tree.ID(feature, "branch-name")
	createBranchID      = tree.ID(feature, "create-branch")
	fetchID             = tree.ID(feature, "fetch")
	pullID              = tree.ID(feature, "pull")
	pushID              = tree.ID(feature, "push")
	pullMergeID         = tree.ID(feature, "pull-policy-merge")
	pullRebaseID        = tree.ID(feature, "pull-policy-rebase")
	pullFFOnlyID        = tree.ID(feature, "pull-policy-ff-only")
	searchID            = tree.ID(feature, "search")
	pathID              = tree.ID(feature, "repository-path")
	browseID            = tree.ID(feature, "browse")
	refreshID           = tree.ID(feature, "refresh")
	branchRefreshID     = tree.ID(feature, "branch-refresh")
	stageID             = tree.ID(feature, "stage")
	unstageID           = tree.ID(feature, "unstage")
	commitID            = tree.ID(feature, "commit")
	messageID           = tree.ID(feature, "commit-message")
	fileScope           = tree.Sub(feature, tree.Name("files"))
	branchScope         = tree.Sub(feature, tree.Name("branch-refs"))
	remoteScope         = tree.Sub(feature, tree.Name("remote-targets"))
	branchSwitchScope   = tree.Sub(feature, tree.Name("branch-switches"))
	fileListID          = tree.ID(feature, "file-list")
	diffScrollID        = tree.ID(feature, "diff-scroll")
	toggleDiffID        = tree.ID(feature, "toggle-diff")
	diffID              = tree.ID(feature, "comparison")
	diffVScrollID       = tree.ID(feature, "comparison-vscroll")
	diffHScrollID       = tree.ID(feature, "comparison-hscroll")
	diffViewportID      = tree.ID(feature, "comparison-viewport")
	hunkScope           = tree.Sub(feature, tree.Name("diff-hunks"))
	amendID             = tree.ID(feature, "amend")
	commitPushID        = tree.ID(feature, "commit-and-push")
	messageHistoryID    = tree.ID(feature, "commit-message-history")
	treeModeID          = tree.ID(feature, "changes-tree-mode")
	flatModeID          = tree.ID(feature, "changes-flat-mode")
	discardID           = tree.ID(feature, "discard")
	discardConfirmID    = tree.ID(feature, "discard-confirm")
	applyHunksID        = tree.ID(feature, "apply-selected-hunks")
	restoreHunksID      = tree.ID(feature, "restore-selected-hunks")
	restoreConfirmID    = tree.ID(feature, "restore-hunks-confirm")
	ignoredToggleID     = tree.ID(feature, "show-ignored")
	errorDetailsID      = tree.ID(feature, "error-details")
	errorScrollID       = tree.ID(feature, "error-scroll")
	logScope            = tree.Sub(feature, tree.Name("commits"))
	logScrollID         = tree.ID(feature, "log-scroll")
	logHeaderScope      = tree.Sub(feature, tree.Name("commit-columns"))
	logDetailsID        = tree.ID(feature, "log-details")
	logFilesID          = tree.ID(feature, "log-files")
	loadMoreID          = tree.ID(feature, "load-more")
	logFilterID         = tree.ID(feature, "log-filter")
	logAuthorID         = tree.ID(feature, "log-author")
	logPathID           = tree.ID(feature, "log-path")
	logSinceID          = tree.ID(feature, "log-since")
	logUntilID          = tree.ID(feature, "log-until")
	logRefID            = tree.ID(feature, "log-ref")
	filterSubmitID      = tree.ID(feature, "filter-submit")
	filterClearID       = tree.ID(feature, "filter-clear")
	commitFileScope     = tree.Sub(feature, tree.Name("commit-files"))
	copySHAID           = tree.ID(feature, "copy-sha")
	commitDiffScrollID  = tree.ID(feature, "commit-diff-scroll")
	commitDiffID        = tree.ID(feature, "commit-comparison")
	commitVScrollID     = tree.ID(feature, "commit-comparison-vscroll")
	commitHScrollID     = tree.ID(feature, "commit-comparison-hscroll")
	commitViewportID    = tree.ID(feature, "commit-comparison-viewport")
	historyTargetID     = tree.ID(feature, "history-target")
	historyPlanID       = tree.ID(feature, "history-plan")
	historyConfirmID    = tree.ID(feature, "history-confirm")
	mergeID             = tree.ID(feature, "history-merge")
	rebaseID            = tree.ID(feature, "history-rebase")
	resetSoftID         = tree.ID(feature, "history-reset-soft")
	resetMixedID        = tree.ID(feature, "history-reset-mixed")
	resetHardID         = tree.ID(feature, "history-reset-hard")
	continueOperationID = tree.ID(feature, "history-continue")
	abortOperationID    = tree.ID(feature, "history-abort")
	skipOperationID     = tree.ID(feature, "history-skip")
	stashMessageID      = tree.ID(feature, "stash-message")
	stashSaveID         = tree.ID(feature, "stash-save")
	stashUntrackedID    = tree.ID(feature, "stash-include-untracked")
	stashBranchNameID   = tree.ID(feature, "stash-branch-name")
	stashDropConfirmID  = tree.ID(feature, "stash-drop-confirm")
	tagNameID           = tree.ID(feature, "tag-name")
	tagTargetID         = tree.ID(feature, "tag-target")
	tagMessageID        = tree.ID(feature, "tag-message")
	settingsLanguageID  = tree.ID(feature, "settings-language")
	settingsPullID      = tree.ID(feature, "settings-pull")
)

type Model struct {
	Language              i18n.Language
	Active                keymap.ActionID
	Git                   git.Installation
	DetectionErr          error
	Checking              bool
	view                  flex.View
	mailbox               workbench.Session
	invalidator           app.Invalidator
	path                  flex.Field
	branchName            flex.Field
	localBranchName       flex.Field
	branchRenameOld       string
	message               commitDraft
	amend                 bool
	commitAndPush         bool
	historyTarget         flex.Field
	historyActionSelect   flex.SelectState
	historyToolsOpen      bool
	historyPlan           *repo.HistoryPlan
	operationConfirm      string
	historyAction         string
	historyRequest        uint64
	messageHistory        []string
	messageLabels         []string
	messageHistorySelect  flex.SelectState
	selectedHistory       int
	treeMode              bool
	discardPlan           *repo.DiscardPlan
	restorePlan           *repo.RestoreHunkPlan
	selectedHunks         []bool
	restoreIgnored        bool
	expandedPaths         map[string]bool
	ignoredPaths          []string
	ignoredRequest        uint64
	messageRequest        uint64
	logQuery              flex.Field
	logAuthor             flex.Field
	logPath               flex.Field
	logSince              flex.Field
	logUntil              flex.Field
	logRef                flex.Field
	logFilter             repo.LogFilter
	repository            *repo.Repository
	status                git.Status
	selected              int
	selectedStaged        bool
	diff                  string
	hunks                 []git.Hunk
	comparison            repo.FileDiff
	aligned               components.CodeDiffResult
	codeDiff              components.CodeDiff
	diffSlot              flex.Box
	showComparison        bool
	busy                  bool
	activeOperation       string
	operationError        string
	showErrorDetails      bool
	request               uint64
	openRequest           uint64
	ctx                   context.Context
	width                 float32
	height                float32
	watchCancel           context.CancelFunc
	watchPending          bool
	commits               []git.Commit
	logTable              flex.TableState
	logSource             commitRows
	logTableBounds        flex.Box
	logColumns            [6]components.TableColumn
	graphRows             []graphRow
	graphWidth            float32
	refs                  []git.Ref
	remotes               []string
	selectedRemote        string
	deletePlan            *repo.BranchDeletePlan
	deletePending         string
	checkoutPlan          *repo.RemoteCheckoutPlan
	confirmForceCheckout  bool
	remoteCancel          context.CancelFunc
	remoteProgress        string
	pullPolicy            repo.PullPolicy
	forcePushPlan         *repo.ForcePushPlan
	pushPlan              *repo.PushPlan
	confirmForcePush      bool
	refsRequest           uint64
	remotesRequest        uint64
	logLoading            bool
	logHasMore            bool
	logRequest            uint64
	detailsRequest        uint64
	selectedCommit        string
	commitMessage         string
	commitFiles           []string
	selectedCommitFile    string
	commitFileRequest     uint64
	commitFileDiff        repo.FileDiff
	commitFileAligned     components.CodeDiffResult
	commitFileReady       bool
	commitDiff            components.CodeDiff
	commitDiffSlot        flex.Box
	changesSplit          flex.SplitterState
	logBranchesSplit      flex.SplitterState
	logDetailsSplit       flex.SplitterState
	logBranchScope        ui.Scope
	settingsStore         *settings.Store
	appSettings           settings.App
	repoSettings          settings.Repository
	stashes               []repo.Stash
	selectedStash         string
	stashDiff             string
	stashMessage          flex.Field
	stashBranchName       flex.Field
	stashIncludeUntracked bool
	stashPending          string
	stashRequest          uint64
	tagName               flex.Field
	tagTarget             flex.Field
	tagMessage            flex.Field
	tagPending            string
}

type commitRows struct {
	commits []git.Commit
	refs    []git.Ref
	head    string
}

func (rows *commitRows) Len() int { return len(rows.commits) }

func (rows *commitRows) Cell(row, column int) string {
	commit := rows.commits[row]
	switch column {
	case 0:
		return ""
	case 1:
		return commit.Subject + refLabels(commit.SHA, rows.refs, rows.head)
	case 2:
		return commit.Author
	case 3:
		return commit.Committer
	case 4:
		if len(commit.Date) >= 16 {
			return commit.Date[5:10] + " " + commit.Date[11:16]
		}
		return commit.Date
	case 5:
		return commit.SHA[:8]
	default:
		return ""
	}
}

var compactCommitColumns = [...]components.TableColumn{{Width: 84}, {}}

func New(language i18n.Language) *Model {
	m := &Model{
		Language: language, Active: keymap.ShowRepositories, Checking: true, selected: -1, ctx: context.Background(),
		changesSplit: flex.SplitterState{Pos: 300}, logBranchesSplit: flex.SplitterState{Pos: 150},
		logDetailsSplit: flex.SplitterState{Pos: 310}, logBranchScope: tree.Sub(feature, tree.Name("log-branches")),
	}
	if store, err := settings.User(); err == nil {
		m.settingsStore = store
		if saved, err := store.LoadApp(); err == nil {
			m.appSettings = saved
			if saved.PullPolicy == "rebase" {
				m.pullPolicy = repo.PullRebase
			} else if saved.PullPolicy == "ff-only" {
				m.pullPolicy = repo.PullFFOnly
			}
			if saved.Language == "zh" {
				m.Language = i18n.Chinese
			}
		}
	}
	return m
}

func (m *Model) SetContext(ctx context.Context) { m.ctx = ctx }

// SetInitialRepository applies a repository loaded before window creation.
func (m *Model) SetInitialRepository(r *repo.Repository, status git.Status, err error) {
	if err != nil {
		m.operationError = err.Error()
		return
	}
	m.repository, m.status, m.Active = r, status, keymap.ShowLog
	if m.settingsStore != nil {
		if saved, err := m.settingsStore.LoadRepository(r.Root); err == nil {
			m.repoSettings = saved
			if saved.ChangesSplit > 0 {
				m.changesSplit.Pos = saved.ChangesSplit
			}
			if saved.LogBranchesSplit > 0 {
				m.logBranchesSplit.Pos = saved.LogBranchesSplit
			}
			if saved.LogDetailsSplit > 0 {
				m.logDetailsSplit.Pos = saved.LogDetailsSplit
			}
			m.treeMode = saved.TreeMode
		}
	}
	m.logQuery.SetText("")
	m.logAuthor.SetText("")
	m.logPath.SetText("")
	m.logSince.SetText("")
	m.logUntil.SetText("")
	m.logRef.SetText("")
	m.logFilter = repo.LogFilter{}
	m.loadMessages()
	m.resetLog()
	m.watch(r)
	m.loadLog()
	m.loadRefs()
	m.loadRemotes()
	m.loadStashes()
}

func (m *Model) saveAppSettings() {
	if m.settingsStore == nil {
		return
	}
	switch m.pullPolicy {
	case repo.PullRebase:
		m.appSettings.PullPolicy = "rebase"
	case repo.PullFFOnly:
		m.appSettings.PullPolicy = "ff-only"
	default:
		m.appSettings.PullPolicy = "merge"
	}
	if m.Language == i18n.Chinese {
		m.appSettings.Language = "zh"
	} else {
		m.appSettings.Language = "en"
	}
	if err := m.settingsStore.SaveApp(m.appSettings); err != nil {
		m.operationError = err.Error()
	}
}

func (m *Model) saveRepositorySettings() {
	if m.settingsStore == nil || m.repository == nil {
		return
	}
	m.repoSettings.ChangesSplit = m.changesSplit.Pos
	m.repoSettings.LogBranchesSplit = m.logBranchesSplit.Pos
	m.repoSettings.LogDetailsSplit = m.logDetailsSplit.Pos
	m.repoSettings.TreeMode = m.treeMode
	if err := m.settingsStore.SaveRepository(m.repository.Root, m.repoSettings); err != nil {
		m.operationError = err.Error()
	}
}

func (m *Model) panelVisible(name string) bool {
	visible, configured := m.repoSettings.PanelVisible[name]
	return !configured || visible
}

func (m *Model) setPanelVisible(name string, visible bool) {
	if m.repoSettings.PanelVisible == nil {
		m.repoSettings.PanelVisible = make(map[string]bool)
	}
	m.repoSettings.PanelVisible[name] = visible
	m.saveRepositorySettings()
}

func (m *Model) loadMessages() {
	if m.repository == nil {
		return
	}
	m.messageRequest++
	m.mailbox.RecentMessages(m.ctx, m.repository, m.messageRequest, m.invalidator.Invalidate)
}

func (m *Model) loadStashes() {
	if m.repository == nil {
		return
	}
	m.stashRequest++
	m.mailbox.Stashes(m.ctx, m.repository, m.stashRequest, m.invalidator.Invalidate)
}

func (m *Model) selectStash(stash repo.Stash) {
	m.selectedStash = stash.SHA
	m.stashDiff = ""
	m.mailbox.StashDiff(m.ctx, m.repository, stash.SHA, m.stashRequest, m.invalidator.Invalidate)
}

func (m *Model) stashWrite(operation, ref, message string) {
	if m.repository == nil || m.busy {
		return
	}
	m.busy, m.activeOperation, m.operationError = true, "stash-"+operation, ""
	m.mailbox.StashWrite(m.ctx, m.repository, operation, ref, message, m.stashIncludeUntracked, m.invalidator.Invalidate)
}

func (m *Model) loadIgnored() {
	if m.repository == nil {
		return
	}
	m.ignoredRequest++
	m.mailbox.Ignored(m.ctx, m.repository, m.ignoredRequest, m.invalidator.Invalidate)
}

func (m *Model) loadRefs() {
	if m.repository == nil {
		return
	}
	m.refsRequest++
	m.mailbox.Refs(m.ctx, m.repository, m.refsRequest, m.invalidator.Invalidate)
}

func (m *Model) loadRemotes() {
	if m.repository == nil {
		return
	}
	m.remotesRequest++
	m.mailbox.Remotes(m.ctx, m.repository, m.remotesRequest, m.invalidator.Invalidate)
}

func (m *Model) watch(r *repo.Repository) {
	if m.watchCancel != nil {
		m.watchCancel()
	}
	m.watchPending = false
	ctx, cancel := context.WithCancel(m.ctx)
	m.watchCancel = cancel
	go func() {
		if err := r.Watch(ctx, func() { m.mailbox.Changed(r, m.invalidator.Invalidate) }, func(err error) {
			m.mailbox.WatchError(r, err, m.invalidator.Invalidate)
		}); err != nil && ctx.Err() == nil {
			m.mailbox.WatchError(r, err, m.invalidator.Invalidate)
		}
	}()
}

func (m *Model) Open(directory string) {
	if m.Git.Path == "" || directory == "" || m.busy {
		return
	}
	m.path.SetText(directory)
	m.busy, m.operationError = true, ""
	m.showErrorDetails = false
	m.openRequest++
	m.mailbox.Open(m.ctx, m.Git.Path, directory, m.openRequest, m.invalidator.Invalidate)
}

func (m *Model) refresh() {
	if m.repository == nil || m.busy {
		return
	}
	m.busy = true
	m.activeOperation = "refresh"
	m.mailbox.Refresh(m.ctx, m.repository, m.invalidator.Invalidate)
}

func (m *Model) loadLog() {
	if m.repository == nil || m.logLoading || !m.logHasMore {
		return
	}
	m.logLoading = true
	m.mailbox.Log(m.ctx, m.repository, len(m.commits), m.logFilter, m.logRequest, m.invalidator.Invalidate)
}

func (m *Model) applyLogFilter() {
	ref := strings.TrimSpace(m.logRef.Text())
	if ref != "" && !strings.HasPrefix(ref, "refs/heads/") && !strings.HasPrefix(ref, "refs/remotes/") {
		ref = "refs/heads/" + ref
	}
	filter := repo.LogFilter{
		Message: strings.TrimSpace(m.logQuery.Text()),
		Author:  strings.TrimSpace(m.logAuthor.Text()),
		Path:    strings.TrimSpace(m.logPath.Text()),
		Ref:     ref,
		Since:   strings.TrimSpace(m.logSince.Text()),
		Until:   strings.TrimSpace(m.logUntil.Text()),
	}
	if filter == m.logFilter {
		return
	}
	m.logFilter = filter
	m.resetLog()
	m.loadLog()
}

func (m *Model) selectCommit(sha string) {
	m.selectedCommit, m.commitMessage, m.commitFiles = sha, "", nil
	for i := range m.commits {
		if m.commits[i].SHA == sha {
			m.logTable.Select(i)
			break
		}
	}
	m.selectedCommitFile, m.commitFileReady = "", false
	m.commitFileRequest++
	m.detailsRequest++
	m.mailbox.CommitDetails(m.ctx, m.repository, sha, m.detailsRequest, m.invalidator.Invalidate)
}

func (m *Model) selectCommitFile(path string) {
	m.selectedCommitFile, m.commitFileReady = path, false
	m.commitFileRequest++
	m.mailbox.CommitFile(m.ctx, m.repository, m.selectedCommit, path, m.commitFileRequest, m.invalidator.Invalidate)
}

func (m *Model) resetLog() {
	m.logRequest++
	m.detailsRequest++
	m.commits, m.selectedCommit, m.commitMessage, m.commitFiles = nil, "", "", nil
	m.logSource.commits = nil
	m.graphRows = nil
	m.logTable.Select(-1)
	m.logTable.Scroll.ScrollTo(0)
	m.logHasMore, m.logLoading = true, false
	m.selectedCommitFile, m.commitFileReady = "", false
	m.commitFileRequest++
}

func (m *Model) selectFile(index int, staged bool) {
	if index < 0 || index >= len(m.status.Files) {
		return
	}
	file := m.status.Files[index]
	m.selected = index
	m.Active = keymap.ShowChanges
	m.selectedStaged = staged
	m.diff = ""
	m.showComparison = false
	m.request++
	m.mailbox.Diff(m.ctx, m.repository, file, m.selectedStaged, m.request, m.invalidator.Invalidate)
}

func (m *Model) write(kind, path, message string) {
	if m.repository == nil || m.busy && m.activeOperation != "refresh" {
		return
	}
	m.busy, m.operationError = true, ""
	m.activeOperation = kind
	m.showErrorDetails = false
	m.mailbox.Write(m.ctx, m.repository, kind, path, message, m.invalidator.Invalidate)
}

func (m *Model) startRemote(operation string) {
	if m.repository == nil || m.busy {
		return
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.remoteCancel = cancel
	m.remoteProgress = m.Language.Text(i18n.Working)
	m.operationError = ""
	m.busy = true
	m.activeOperation = "remote-" + operation
	m.mailbox.Remote(ctx, m.repository, operation, m.selectedRemote, m.pullPolicy, m.invalidator.Invalidate)
}

func (m *Model) planPush() {
	if m.repository == nil {
		return
	}
	m.busy = true
	m.activeOperation = "push-plan"
	m.operationError = ""
	m.mailbox.PlanPush(m.ctx, m.repository, m.selectedRemote, m.invalidator.Invalidate)
}

func (m *Model) planHistory(action string) {
	if m.repository == nil || m.busy {
		return
	}
	target := strings.TrimSpace(m.historyTarget.Text())
	if target == "" {
		return
	}
	m.historyAction = action
	m.historyRequest++
	m.busy, m.operationError = true, ""
	m.activeOperation = "history-plan"
	m.mailbox.PlanHistory(m.ctx, m.repository, action, target, m.historyRequest, m.invalidator.Invalidate)
}

func (m *Model) resolveOperation(action string) {
	if m.repository == nil || m.busy {
		return
	}
	if action == "abort" || action == "skip" {
		m.operationConfirm = action
		return
	}
	m.executeOperation(action)
}

func (m *Model) executeOperation(action string) {
	m.busy, m.operationError = true, ""
	m.activeOperation = "history-operation"
	m.mailbox.ResolveOperation(m.ctx, m.repository, action, m.invalidator.Invalidate)
}

// SetInitialGit applies startup detection before the window thread begins.
func (m *Model) SetInitialGit(installation git.Installation, err error) {
	m.Git, m.DetectionErr, m.Checking = installation, err, false
}

func (m *Model) Start(ctx context.Context, gitPath string) {
	m.mailbox.DetectGit(ctx, gitPath, m.invalidator.Invalidate)
}

func (m *Model) Config() app.Config {
	text := m.Language.Text
	theme := ui.DarkTheme()
	theme.Metrics = theme.Metrics.WithDensity(ui.DensityCompact)
	theme.Background = ui.RGB(25, 27, 31)
	theme.Card = ui.RGB(29, 31, 36)
	theme.CardElevated = ui.RGB(35, 38, 44)
	theme.Muted = ui.RGB(41, 44, 51)
	theme.Hover = ui.RGB(47, 51, 59)
	theme.Border = ui.RGB(52, 56, 64)
	theme.Primary = ui.RGB(53, 91, 158)
	theme.Selection = ui.RGB(44, 70, 116)
	theme.MutedForeground = ui.RGB(154, 163, 178)
	config := app.Config{
		Tree: &tree, Title: text(i18n.AppName), Width: 1200, Height: 760,
		MinWidth: 720, MinHeight: 480, Invalidator: &m.invalidator,
		TextInput: func(context app.TextInputContext) *app.TextInput {
			if context.ID == messageID {
				m.message.ensure()
				return &m.message.input
			}
			return flex.Fields(&m.path, &m.branchName, &m.localBranchName, &m.historyTarget, &m.logQuery, &m.logAuthor, &m.logPath, &m.logSince, &m.logUntil, &m.logRef, &m.stashMessage, &m.stashBranchName, &m.tagName, &m.tagTarget, &m.tagMessage)(context)
		},
		OnScroll: func(event ui.ScrollEvent) ui.EventResult {
			if m.Active == keymap.ShowLog && contains(m.logTableBounds.Rect(), event.X, event.Y) {
				result := m.logTable.Wheel(event)
				if m.logHasMore && m.logTable.Scroll.Offset >= m.logTable.Scroll.Max()-125 {
					m.loadLog()
				}
				return result
			}
			if m.showComparison && m.Active == keymap.ShowChanges && m.containsDiff(event.X, event.Y) {
				return m.codeDiff.Scroll(event)
			}
			if m.commitFileReady && !m.commitFileDiff.Binary && m.Active == keymap.ShowLog && contains(m.commitDiffSlot.Rect(), event.X, event.Y) {
				return m.commitDiff.Scroll(event)
			}
			return m.view.Wheel(event)
		},
		OnKey: func(event ui.KeyEvent) ui.EventResult {
			for _, split := range []*flex.SplitterState{&m.changesSplit, &m.logBranchesSplit, &m.logDetailsSplit} {
				if result := split.Key(event); result.Handled {
					m.saveRepositorySettings()
					return result
				}
			}
			if event.Target == messageID {
				return m.message.move(event)
			}
			if m.Active == keymap.ShowLog {
				result := m.logTable.Key(event)
				if result.Handled && m.logTable.Selected() >= 0 && m.logTable.Selected() < len(m.commits) {
					m.selectCommit(m.commits[m.logTable.Selected()].SHA)
				}
				return result
			}
			return ui.EventResult{}
		},
		OnShortcut: func(event ui.KeyEvent) ui.EventResult {
			if event.Modifiers&(ui.ModifierControl|ui.ModifierMeta) == 0 || event.Text == "" || event.Target == messageID || event.Target == stashMessageID || event.Target == stashBranchNameID {
				return ui.EventResult{}
			}
			switch strings.ToLower(event.Text) {
			case "1":
				m.Active = keymap.ShowChanges
			case "2":
				m.Active = keymap.ShowLog
			case "3":
				m.Active = keymap.ShowBranches
			case "4":
				m.Active = keymap.ShowStash
				m.loadStashes()
			case "t":
				m.Active = keymap.ShowTags
			case "r":
				m.refresh()
			default:
				return ui.EventResult{}
			}
			return ui.EventResult{Handled: true, Repaint: true}
		},
		OnPointer: func(event ui.PointerEvent) ui.EventResult {
			if event.Kind == ui.PointerUp && (event.Target == tree.ID(feature, "changes-split") || event.Target == tree.ID(feature, "log-branches-split") || event.Target == tree.ID(feature, "log-details-split")) {
				m.saveRepositorySettings()
			}
			if m.Active == keymap.ShowLog && m.selectedCommit != "" && event.Kind == ui.PointerDown && event.Target == copySHAID && event.Button == ui.PointerButtonPrimary {
				return ui.EventResult{Handled: true, ClipboardText: m.selectedCommit}
			}
			if event.Kind == ui.PointerDown && event.Target == browseID && event.Button == ui.PointerButtonPrimary {
				if m.busy {
					return ui.EventResult{Handled: true}
				}
				path, err := app.OpenDirectory(app.OpenDirectoryOptions{Title: text(i18n.OpenRepository)})
				switch {
				case err == nil:
					m.Open(path)
				case err != app.ErrCanceled:
					m.operationError = err.Error()
				}
				return ui.EventResult{Handled: true, Repaint: true}
			}
			if m.showComparison && m.Active == keymap.ShowChanges {
				return m.codeDiff.Pointer(event)
			}
			if m.commitFileReady && !m.commitFileDiff.Binary && m.Active == keymap.ShowLog {
				return m.commitDiff.Pointer(event)
			}
			return ui.EventResult{}
		},
		OnWake: func() {
			for _, result := range m.mailbox.Drain() {
				m.Git, m.DetectionErr, m.Checking = result.Installation, result.Err, false
			}
			for _, update := range m.mailbox.DrainUpdates() {
				if update.Kind == "open" && update.Request != m.openRequest || update.Kind != "open" && update.Repository != m.repository {
					continue
				}
				if update.Kind == "diff" && update.Request != m.request || update.Kind == "commit-details" && (update.Request != m.detailsRequest || update.Path != m.selectedCommit) || update.Kind == "log" && update.Request != m.logRequest {
					continue
				}
				if update.Kind == "refs" && update.Request != m.refsRequest {
					continue
				}
				if update.Kind == "remotes" && update.Request != m.remotesRequest {
					continue
				}
				if update.Kind == "commit-messages" && update.Request != m.messageRequest {
					continue
				}
				if update.Kind == "ignored" && update.Request != m.ignoredRequest {
					continue
				}
				if update.Kind == "discard-plan" && update.Request != m.request {
					continue
				}
				if update.Kind == "history-plan" && update.Request != m.historyRequest {
					continue
				}
				if update.Kind == "refresh" && m.activeOperation != "refresh" {
					continue
				}
				if update.Kind == "commit-file" && (update.Request != m.commitFileRequest || update.Message != m.selectedCommit || update.Path != m.selectedCommitFile) {
					continue
				}
				if update.Err != nil {
					m.operationError = update.Err.Error()
				}
				switch update.Kind {
				case "open":
					if update.Err == nil {
						m.repository, m.status = update.Repository, update.Status
						if m.settingsStore != nil {
							if saved, err := m.settingsStore.LoadRepository(update.Repository.Root); err == nil {
								m.repoSettings = saved
								if saved.ChangesSplit > 0 {
									m.changesSplit.Pos = saved.ChangesSplit
								}
								if saved.LogBranchesSplit > 0 {
									m.logBranchesSplit.Pos = saved.LogBranchesSplit
								}
								if saved.LogDetailsSplit > 0 {
									m.logDetailsSplit.Pos = saved.LogDetailsSplit
								}
								m.treeMode = saved.TreeMode
							}
						}
						m.logQuery.SetText("")
						m.logAuthor.SetText("")
						m.logPath.SetText("")
						m.logSince.SetText("")
						m.logUntil.SetText("")
						m.logRef.SetText("")
						m.logFilter = repo.LogFilter{}
						m.resetLog()
						m.refs = nil
						m.remotes = nil
						m.selectedRemote = ""
						m.watch(update.Repository)
						m.path.SetText("")
						m.Active, m.selected, m.diff, m.showComparison = keymap.ShowLog, -1, "", false
						m.loadMessages()
						if m.restoreIgnored {
							m.loadIgnored()
						}
						m.loadLog()
						m.loadRefs()
						m.loadRemotes()
						m.loadStashes()
					}
				case "diff":
					if update.Err == nil && m.selected >= 0 && m.selected < len(m.status.Files) && m.status.Files[m.selected].Path == update.Path && m.selectedStaged == update.Staged {
						m.diff = update.Diff
						m.comparison, m.aligned = update.Comparison, update.Aligned
						m.hunks, _ = git.ParseHunks([]byte(update.Diff))
						m.selectedHunks = make([]bool, len(m.hunks))
						m.showComparison = !update.Comparison.Binary
					}
				case "watch-error":
					m.operationError = update.Err.Error()
				case "log":
					if update.Request != m.logRequest {
						break
					}
					m.logLoading = false
					if update.Err == nil && update.Skip == len(m.commits) {
						m.commits = append(m.commits, update.Commits...)
						m.logSource.commits = m.commits
						m.graphRows = buildGraph(m.commits)
						m.graphWidth = 84
						for _, row := range m.graphRows {
							m.graphWidth = max(m.graphWidth, 24+float32(max(len(row.before), len(row.after)))*12)
						}
						m.logHasMore = len(update.Commits) == 100
						if m.selectedCommit == "" && len(m.commits) > 0 {
							m.selectCommit(m.commits[0].SHA)
						}
					}
				case "commit-details":
					if update.Err == nil && update.Request == m.detailsRequest && update.Path == m.selectedCommit {
						m.commitMessage, m.commitFiles = update.Message, update.Files
					}
				case "commit-messages":
					if update.Err == nil {
						m.messageHistory = update.Messages
						m.messageLabels = make([]string, len(update.Messages))
						for i, value := range update.Messages {
							m.messageLabels[i] = strings.SplitN(strings.ReplaceAll(value, "\r", ""), "\n", 2)[0]
						}
					}
				case "discard-plan":
					if update.Err == nil {
						m.discardPlan = &update.DiscardPlan
					}
				case "discard":
					m.discardPlan = nil
				case "restore-hunks-plan":
					if update.Err == nil {
						m.restorePlan = &update.RestorePlan
					}
				case "restore-hunks":
					m.restorePlan = nil
				case "history-plan":
					if update.Err == nil {
						m.historyPlan = &update.HistoryPlan
					}
					m.busy = false
					m.activeOperation = ""
				case "history-operation":
					m.historyPlan = nil
					m.operationConfirm = ""
					if update.Status.Root != "" {
						m.status = update.Status
						m.loadRefs()
						m.selected, m.diff, m.showComparison = -1, "", false
						m.resetLog()
						m.loadLog()
					}
				case "stashes":
					if update.Request == m.stashRequest && update.Err == nil {
						m.stashes = update.Stashes
						if !slices.ContainsFunc(m.stashes, func(stash repo.Stash) bool { return stash.SHA == m.selectedStash }) {
							m.selectedStash, m.stashDiff = "", ""
						}
						if m.selectedStash == "" && len(m.stashes) > 0 {
							m.selectStash(m.stashes[0])
						}
					}
				case "stash-diff":
					if update.Err == nil && update.Path == m.selectedStash {
						m.stashDiff = update.Message
					}
				case "stash-save", "stash-apply", "stash-pop", "stash-drop", "stash-branch":
					m.stashPending = ""
					if update.Status.Root != "" {
						m.status = update.Status
					}
					m.loadStashes()
					m.loadRefs()
					m.resetLog()
					m.loadLog()
				case "tag-create", "tag-delete":
					m.tagPending = ""
					m.loadRefs()
					m.resetLog()
					m.loadLog()
				case "resolve-conflict":
					if update.Status.Root != "" {
						m.status = update.Status
					}
					m.selected, m.diff, m.showComparison = -1, "", false
				case "ignored":
					if update.Err == nil {
						m.ignoredPaths = update.Paths
					}
				case "commit-file":
					if update.Err == nil {
						m.commitFileDiff, m.commitFileAligned, m.commitFileReady = update.Comparison, update.Aligned, true
					}
				case "refs":
					if update.Err == nil {
						m.refs = update.Refs
						m.logSource.refs, m.logSource.head = m.refs, m.status.Head
						for _, ref := range m.refs {
							if !ref.Remote && ref.Name == m.status.Branch && ref.Track != "" {
								m.status.Ahead = strings.Count(ref.Track, ">")
								m.status.Behind = strings.Count(ref.Track, "<")
								m.status.HasTrack = true
								break
							}
						}
					}
				case "remotes":
					if update.Err == nil {
						m.remotes = update.Remotes
						if m.selectedRemote != "" && !slices.Contains(m.remotes, m.selectedRemote) {
							m.selectedRemote = ""
						}
					}
				case "create-branch":
					if update.Err == nil {
						m.branchName.SetText("")
						m.loadRefs()
						m.resetLog()
						m.loadLog()
					}
				case "switch-branch":
					if update.Status.Root != "" {
						m.status = update.Status
					}
				case "branch-delete-plan":
					if update.Err == nil {
						m.deletePlan = &update.BranchPlan
					}
					m.busy = false
					m.activeOperation = ""
				case "remote-checkout-plan":
					if update.Err == nil {
						m.checkoutPlan = &update.CheckoutPlan
						m.confirmForceCheckout = false
					}
					m.busy = false
					m.activeOperation = ""
				case "push-plan":
					if update.Err == nil {
						m.pushPlan = &update.PushPreview
					}
					m.busy = false
					m.activeOperation = ""
				case "force-push-plan":
					if update.Err == nil {
						m.forcePushPlan = &update.PushPlan
						m.confirmForcePush = false
					}
					m.busy = false
					m.activeOperation = ""
				case "checkout-remote":
					if update.Status.Root != "" {
						m.status = update.Status
					}
					if update.Err == nil {
						m.checkoutPlan = nil
						m.confirmForceCheckout = false
						m.selected, m.diff, m.showComparison = -1, "", false
						m.request++
						m.loadRefs()
						m.resetLog()
						m.loadLog()
					}
				case "rename-branch", "set-upstream", "delete-branch":
					if update.Err == nil {
						m.loadRefs()
						if update.Kind == "rename-branch" {
							m.branchRenameOld = ""
							m.localBranchName.SetText("")
							if m.logFilter.Ref == "refs/heads/"+update.Path {
								m.logFilter.Ref = ""
							}
						}
						if update.Kind == "delete-branch" {
							m.deletePlan = nil
						}
						m.selected, m.diff, m.showComparison = -1, "", false
						m.request++
						m.resetLog()
						m.loadLog()
					}
				case "changed":
					m.watchPending = true
				case "remote-progress":
					m.remoteProgress = update.Message
				case "remote-fetch", "remote-pull", "remote-push":
					if m.remoteCancel != nil {
						m.remoteCancel()
						m.remoteCancel = nil
					}
					m.remoteProgress = ""
					if update.Status.Root != "" {
						m.status = update.Status
					}
					if update.Err == nil {
						m.loadRefs()
						m.resetLog()
						m.loadLog()
						m.forcePushPlan = nil
						m.confirmForcePush = false
					}
				default:
					if update.Kind == "stage" || update.Kind == "unstage" || update.Kind == "stage-hunk" || update.Kind == "unstage-hunk" || update.Kind == "commit" || update.Kind == "amend" || update.Kind == "create-branch" || update.Kind == "switch-branch" {
						m.watchPending = false
					}
					if update.Status.Root != "" {
						unchanged := update.Kind == "refresh" && m.status.Root == update.Status.Root && m.status.Head == update.Status.Head && m.status.Branch == update.Status.Branch && m.status.Ahead == update.Status.Ahead && m.status.Behind == update.Status.Behind && m.status.HasTrack == update.Status.HasTrack && m.status.Operation == update.Status.Operation && slices.Equal(m.status.Files, update.Status.Files)
						m.status = update.Status
						if !unchanged {
							m.diff = ""
							m.showComparison = false
							m.selected = -1
							m.request++
						}
					}
					if (update.Kind == "commit" || update.Kind == "amend") && update.Err == nil {
						m.message.SetText("")
						m.amend = false
						m.resetLog()
						m.loadRefs()
						m.loadMessages()
						if m.commitAndPush {
							m.commitAndPush = false
							m.planPush()
						}
					} else if update.Kind == "commit" || update.Kind == "amend" {
						m.commitAndPush = false
					}
				}
				if update.Kind != "diff" && update.Kind != "changed" && update.Kind != "watch-error" && update.Kind != "log" && update.Kind != "commit-details" && update.Kind != "commit-file" && update.Kind != "refs" && update.Kind != "remotes" && update.Kind != "remote-progress" {
					if m.activeOperation == "" || update.Kind == m.activeOperation {
						m.busy = false
						m.activeOperation = ""
					}
				}
			}
			if m.watchPending && !m.busy && m.repository != nil {
				m.watchPending = false
				m.refresh()
			}
		},
		OnClosed: func() {
			if m.watchCancel != nil {
				m.watchCancel()
			}
			m.saveRepositorySettings()
		},
	}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		m.width = size.Width
		m.height = size.Height
		bounds := ui.Rect{Width: size.Width, Height: size.Height}
		canvas.Fill(bounds, theme.Background)
		frame := m.view.Begin(canvas, theme, bounds)
		func(f *flex.Frame) {
			f.Column().Fill().Children(func() {
				m.paintTop(f)
				f.Divider()
				f.Row().FillWidth().Grow(1).Children(func() {
					m.paintNavigation(f)
					f.Divider()
					if m.repository != nil && m.Active != keymap.ShowRepositories {
						changesWidth := min(max(m.changesSplit.Pos, float32(240)), max(float32(240), m.width*0.34))
						f.Column().Width(changesWidth).FillHeight().Children(func() { m.paintChangesSidebar(f) })
						f.Splitter(tree.ID(feature, "changes-split"), &m.changesSplit).Mins(230, 300)
					}
					m.paintContent(f)
				})
				f.Divider()
				m.paintStatus(f)
			})
		}(frame)
		frame.End()
		if m.Active == keymap.ShowLog && len(m.graphRows) > 0 {
			paintGraph(canvas, m.logTableBounds.Rect(), m.logTable.Scroll.Offset, m.graphRows)
		}
		if m.showComparison && m.Active == keymap.ShowChanges {
			m.codeDiff.Paint(canvas, theme, m.diffSlot.Rect(), components.CodeDiffSnapshot{
				Result: m.aligned, ID: diffID, Label: text(i18n.StagedDiff), Value: m.comparison.Right,
				ReadOnly: true, FontSize: 12, VerticalScrollbarID: diffVScrollID,
				HorizontalScrollbarID: diffHScrollID, ViewportID: diffViewportID,
			})
		}
		if m.Active == keymap.ShowLog && m.commitFileReady && !m.commitFileDiff.Binary {
			m.commitDiff.Paint(canvas, theme, m.commitDiffSlot.Rect(), components.CodeDiffSnapshot{
				Result: m.commitFileAligned, ID: commitDiffID, Label: m.selectedCommitFile,
				Value: m.commitFileDiff.Right, ReadOnly: true, FontSize: 12,
				VerticalScrollbarID: commitVScrollID, HorizontalScrollbarID: commitHScrollID,
				ViewportID: commitViewportID,
			})
		}
	}
	return config
}

func (m *Model) paintTop(f *flex.Frame) {
	text := m.Language.Text
	f.Row().FillWidth().Height(40).PadX(10).Gap(9).Items(flex.Center).Background(ui.RGB(34, 36, 42)).Children(func() {
		f.Text("R").Color(ui.RGB(220, 122, 66)).Bold().FontSize(15)
		f.Text(text(i18n.AppName)).FontSize(12)
		f.Divider().Height(16)
		label := text(i18n.OpenRepository)
		if m.repository != nil {
			label = filepath.Base(m.status.Root)
		}
		if f.Button(openID, label).Ghost().Small().Clicked() {
			m.Active = keymap.ShowRepositories
		}
		branch := text(i18n.NoBranch)
		if m.repository != nil {
			branch = m.status.Branch
			if m.status.HasTrack {
				branch += fmt.Sprintf("  ↑%d ↓%d", m.status.Ahead, m.status.Behind)
			}
		}
		if f.Button(branchID, branch).Ghost().Small().Disabled(m.repository == nil).Clicked() {
			m.Active = keymap.ShowBranches
			m.loadRefs()
		}
		f.Spacer()
		if m.width >= 900 {
			if m.busy && m.remoteCancel != nil {
				if f.Button(fetchID, text(i18n.Cancel)).Ghost().Small().Clicked() {
					m.remoteCancel()
				}
			} else {
				if f.Button(fetchID, text(i18n.Fetch)).Ghost().Small().Disabled(m.repository == nil || m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
					m.startRemote("fetch")
				}
				f.Text(text(i18n.PullPolicyLabel)).Muted().FontSize(11)
				mergeLabel := text(i18n.PullMerge)
				if m.pullPolicy == repo.PullMerge {
					mergeLabel += " *"
				}
				if f.Button(pullMergeID, mergeLabel).Ghost().Small().Disabled(m.busy).Clicked() {
					m.pullPolicy = repo.PullMerge
					m.saveAppSettings()
				}
				rebaseLabel := text(i18n.PullRebase)
				if m.pullPolicy == repo.PullRebase {
					rebaseLabel += " *"
				}
				if f.Button(pullRebaseID, rebaseLabel).Ghost().Small().Disabled(m.busy).Clicked() {
					m.pullPolicy = repo.PullRebase
					m.saveAppSettings()
				}
				ffOnlyLabel := text(i18n.PullFFOnly)
				if m.pullPolicy == repo.PullFFOnly {
					ffOnlyLabel += " *"
				}
				if f.Button(pullFFOnlyID, ffOnlyLabel).Ghost().Small().Disabled(m.busy).Clicked() {
					m.pullPolicy = repo.PullFFOnly
					m.saveAppSettings()
				}
				if f.Button(pullID, text(i18n.Pull)).Ghost().Small().Disabled(m.repository == nil || m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
					m.startRemote("pull")
				}
				if f.Button(pushID, text(i18n.Push)).Ghost().Small().Disabled(m.repository == nil || m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
					m.Active = keymap.ShowBranches
					m.planPush()
				}
				if f.Button(tree.ID(feature, "force-push"), text(i18n.ForcePushWithLease)).Ghost().Small().Disabled(m.repository == nil || m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
					m.Active = keymap.ShowBranches
					m.busy = true
					m.activeOperation = "force-push-plan"
					m.operationError = ""
					m.mailbox.PlanForcePush(m.ctx, m.repository, m.selectedRemote, m.invalidator.Invalidate)
				}
			}
			f.Button(searchID, text(i18n.Search)).Ghost().Small().Disabled(true)
		}
	})
}

func (m *Model) paintNavigation(f *flex.Frame) {
	icons := [...]ui.Icon{GitCommitHorizontal, List, GitBranch, Archive, Tag, Folder, Tag}
	f.Column().Width(44).FillHeight().Pad(5).Gap(5).Background(ui.RGB(34, 36, 42)).Children(func() {
		for i, action := range keymap.Actions {
			background := ui.RGB(34, 36, 42)
			if m.Active == action.ID {
				background = ui.RGB(53, 57, 66)
			}
			label := m.Language.Text(action.Name)
			row := f.Row().FillWidth().Height(34).Center().Radius(4).Background(background).
				HoverBackground(ui.RGB(58, 62, 71)).HitWith(ids[i], label, ui.HitOptions{
				Role: ui.RoleButton, Cursor: ui.PointerCursorHand, Hint: label,
			}).Children(func() {
				f.Icon(icons[i]).Size(17, 17)
			})
			if row.Clicked() {
				m.Active = action.ID
				if action.ID == keymap.ShowStash {
					m.loadStashes()
				}
				if action.ID == keymap.ShowLog && m.repository != nil && m.logHasMore && len(m.commits) == 0 {
					m.loadLog()
				}
			}
		}
	})
}

func (m *Model) paintContent(f *flex.Frame) {
	text := m.Language.Text
	if m.Active == keymap.ShowSettings {
		m.paintSettings(f)
		return
	}
	if m.Active == keymap.ShowRepositories || m.repository == nil {
		m.paintRepository(f)
		return
	}
	if m.Active == keymap.ShowChanges {
		m.paintChangesDiff(f)
		return
	}
	if m.Active == keymap.ShowLog {
		m.paintLog(f)
		return
	}
	if m.Active == keymap.ShowBranches {
		m.paintBranches(f)
		return
	}
	if m.Active == keymap.ShowStash {
		m.paintStashes(f)
		return
	}
	if m.Active == keymap.ShowTags {
		m.paintTags(f)
		return
	}
	f.Column().Fill().Pad(16).Gap(10).Children(func() {
		f.Text(text(keymap.Actions[m.Active].Name)).FontSize(15).Bold()
		f.Divider()
		f.Spacer()
		f.Row().FillWidth().Justify(flex.Center).Children(func() {
			f.Text(text(i18n.SelectRepository)).Muted()
		})
		f.Spacer()
	})
}

func (m *Model) paintSettings(f *flex.Frame) {
	text := m.Language.Text
	f.Column().Fill().Pad(18).Gap(10).Children(func() {
		f.Text(text(i18n.Settings)).Bold().FontSize(15)
		f.Divider()
		f.Row().FillWidth().Height(34).Items(flex.Center).Gap(10).Children(func() {
			f.Text(text(i18n.LanguageSetting)).Width(180).FontSize(12)
			english := text(i18n.LanguageSetting) + ": English"
			chinese := text(i18n.LanguageSetting) + ": 简体中文"
			if f.Button(settingsLanguageID, english).Ghost().Small().Clicked() {
				m.Language = i18n.English
				m.saveAppSettings()
			}
			if f.Button(tree.ID(feature, "settings-language-zh"), chinese).Ghost().Small().Clicked() {
				m.Language = i18n.Chinese
				m.saveAppSettings()
			}
		})
		f.Row().FillWidth().Height(34).Items(flex.Center).Gap(10).Children(func() {
			f.Text(text(i18n.PullDefault)).Width(180).FontSize(12)
			for index, key := range []i18n.Key{i18n.PullMerge, i18n.PullRebase, i18n.PullFFOnly} {
				label := text(key)
				if int(m.pullPolicy) == index {
					label += " *"
				}
				id := tree.ID(feature, []string{"settings-pull-merge", "settings-pull-rebase", "settings-pull-ff-only"}[index])
				if f.Button(id, label).Ghost().Small().Clicked() {
					m.pullPolicy = repo.PullPolicy(index)
					m.saveAppSettings()
				}
			}
		})
		f.Text(text(i18n.Theme)).Muted().FontSize(11)
	})
}

func (m *Model) paintBranches(f *flex.Frame) {
	text := m.Language.Text
	f.Column().Fill().Pad(14).Gap(8).Children(func() {
		f.Row().FillWidth().Items(flex.Center).Children(func() {
			f.Text(text(i18n.Branches)).Bold().FontSize(15)
			f.Spacer()
			if f.Button(branchRefreshID, text(i18n.Refresh)).Ghost().Small().Clicked() {
				m.loadRefs()
			}
		})
		f.Row().FillWidth().Gap(8).Children(func() {
			if f.Field(&m.branchName, branchNameID, text(i18n.BranchName), text(i18n.BranchName)).Grow(1).Submitted() {
				m.write("create-branch", strings.TrimSpace(m.branchName.Text()), "")
			}
			if f.Button(createBranchID, text(i18n.NewBranch)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite || strings.TrimSpace(m.branchName.Text()) == "").Clicked() {
				m.write("create-branch", strings.TrimSpace(m.branchName.Text()), "")
			}
		})
		if m.deletePlan != nil {
			f.Row().FillWidth().Gap(8).Items(flex.Center).Children(func() {
				f.Text(fmt.Sprintf("%s %s (%s)", text(i18n.ConfirmDeleteBranch), m.deletePlan.Name, m.deletePlan.Target[:min(8, len(m.deletePlan.Target))])).FontSize(12)
				if f.Button(tree.ID(feature, "confirm-delete-branch"), text(i18n.DeleteBranch)).Small().Disabled(m.busy).Clicked() {
					plan := *m.deletePlan
					m.deletePlan = nil
					m.busy = true
					m.activeOperation = "delete-branch"
					m.mailbox.DeleteBranch(m.ctx, m.repository, plan, m.invalidator.Invalidate)
				}
				if f.Button(tree.ID(feature, "cancel-delete-branch"), text(i18n.Cancel)).Ghost().Small().Clicked() {
					m.deletePlan = nil
				}
			})
		}
		if m.checkoutPlan != nil {
			f.Column().FillWidth().Gap(5).Pad(8).Background(ui.RGB(48, 43, 36)).Children(func() {
				f.Text(fmt.Sprintf("%s %s -> %s", text(i18n.CheckoutChangesWarning), m.checkoutPlan.Remote, m.checkoutPlan.Local)).FontSize(12)
				if len(m.checkoutPlan.DirtyFiles) == 0 {
					f.Text(text(i18n.Ready)).Muted().FontSize(11)
				} else {
					f.ScrollArea(tree.ID(feature, "checkout-conflicting-files")).Height(min(float32(96), float32(len(m.checkoutPlan.DirtyFiles))*19)).Children(func() {
						for _, path := range m.checkoutPlan.DirtyFiles {
							f.Text(path).Mono().FontSize(11)
						}
					})
				}
				f.Row().FillWidth().Gap(6).Children(func() {
					if len(m.checkoutPlan.DirtyFiles) == 0 {
						if f.Button(tree.ID(feature, "checkout-confirm"), text(i18n.CheckoutRemoteBranch)).Small().Disabled(m.busy).Clicked() {
							plan := *m.checkoutPlan
							m.busy = true
							m.activeOperation = "checkout-remote"
							m.mailbox.ExecuteRemoteCheckout(m.ctx, m.repository, plan, repo.CheckoutSafe, m.invalidator.Invalidate)
						}
					} else {
						if f.Button(tree.ID(feature, "checkout-stash"), text(i18n.StashAndCheckout)).Small().Disabled(m.busy).Clicked() {
							plan := *m.checkoutPlan
							m.checkoutPlan = nil
							m.busy = true
							m.activeOperation = "checkout-remote"
							m.mailbox.ExecuteRemoteCheckout(m.ctx, m.repository, plan, repo.CheckoutStash, m.invalidator.Invalidate)
						}
						forceLabel := text(i18n.ForceCheckout)
						if m.confirmForceCheckout {
							forceLabel = text(i18n.ConfirmForceCheckout)
						}
						if f.Button(tree.ID(feature, "checkout-force"), forceLabel).Small().Disabled(m.busy).Clicked() {
							if !m.confirmForceCheckout {
								m.confirmForceCheckout = true
							} else {
								plan := *m.checkoutPlan
								m.checkoutPlan = nil
								m.confirmForceCheckout = false
								m.busy = true
								m.activeOperation = "checkout-remote"
								m.mailbox.ExecuteRemoteCheckout(m.ctx, m.repository, plan, repo.CheckoutForce, m.invalidator.Invalidate)
							}
						}
					}
					if f.Button(tree.ID(feature, "checkout-cancel"), text(i18n.Cancel)).Ghost().Small().Clicked() {
						m.checkoutPlan = nil
						m.confirmForceCheckout = false
						m.activeOperation = ""
					}
				})
			})
		}
		if m.forcePushPlan != nil {
			f.Column().FillWidth().Gap(5).Pad(8).Background(ui.RGB(48, 43, 36)).Children(func() {
				f.Text(text(i18n.PushTargets)).FontSize(12)
				f.Text(fmt.Sprintf("%s/%s  %s -> %s", m.forcePushPlan.Remote, m.forcePushPlan.Branch, m.forcePushPlan.Local[:min(8, len(m.forcePushPlan.Local))], m.forcePushPlan.Lease[:min(8, len(m.forcePushPlan.Lease))])).Mono().FontSize(11)
				f.Row().FillWidth().Gap(6).Children(func() {
					label := text(i18n.ForcePushWithLease)
					if m.confirmForcePush {
						label = text(i18n.ConfirmForcePush)
					}
					if f.Button(tree.ID(feature, "force-push-confirm"), label).Small().Disabled(m.busy).Clicked() {
						if !m.confirmForcePush {
							m.confirmForcePush = true
						} else {
							plan := *m.forcePushPlan
							ctx, cancel := context.WithCancel(m.ctx)
							m.remoteCancel = cancel
							m.remoteProgress = text(i18n.Working)
							m.forcePushPlan = nil
							m.confirmForcePush = false
							m.busy = true
							m.activeOperation = "remote-push"
							m.mailbox.ForcePush(ctx, m.repository, plan, m.invalidator.Invalidate)
						}
					}
					if f.Button(tree.ID(feature, "force-push-cancel"), text(i18n.Cancel)).Ghost().Small().Clicked() {
						m.forcePushPlan = nil
						m.confirmForcePush = false
					}
				})
			})
		}
		if m.pushPlan != nil {
			f.Column().FillWidth().Gap(5).Pad(8).Background(ui.RGB(43, 48, 54)).Children(func() {
				f.Text(fmt.Sprintf("%s %s/%s  %s -> %s", text(i18n.PushPreview), m.pushPlan.Remote, m.pushPlan.Branch, m.pushPlan.Local[:min(8, len(m.pushPlan.Local))], m.pushPlan.Upstream[:min(8, len(m.pushPlan.Upstream))])).Mono().FontSize(11)
				for _, commit := range m.pushPlan.Commits[:min(5, len(m.pushPlan.Commits))] {
					f.Text(fmt.Sprintf("%s %s", commit.SHA[:8], commit.Subject)).FontSize(11).Ellipsis()
				}
				if len(m.pushPlan.Commits) > 5 {
					f.Text(fmt.Sprintf("+%d", len(m.pushPlan.Commits)-5)).Muted().FontSize(11)
				}
				f.Row().FillWidth().Gap(6).Children(func() {
					if f.Button(tree.ID(feature, "push-confirm"), text(i18n.Push)).Small().Disabled(m.busy).Clicked() {
						plan := *m.pushPlan
						m.pushPlan = nil
						ctx, cancel := context.WithCancel(m.ctx)
						m.remoteCancel = cancel
						m.remoteProgress = text(i18n.Working)
						m.busy = true
						m.activeOperation = "remote-push"
						m.mailbox.ExecutePush(ctx, m.repository, plan, m.invalidator.Invalidate)
					}
					if f.Button(tree.ID(feature, "push-cancel"), text(i18n.Cancel)).Ghost().Small().Clicked() {
						m.pushPlan = nil
					}
				})
			})
		}
		if len(m.remotes) > 1 {
			f.Row().FillWidth().Gap(5).PadX(8).Items(flex.Center).Children(func() {
				f.Text(text(i18n.RemoteTarget)).Muted().FontSize(11)
				upstreamLabel := text(i18n.UpstreamDefault)
				if m.selectedRemote == "" {
					upstreamLabel += " *"
				}
				if f.Button(tree.ID(feature, "remote-upstream"), upstreamLabel).Ghost().Small().Clicked() {
					m.selectedRemote = ""
				}
				for i, remote := range m.remotes {
					label := remote
					if remote == m.selectedRemote {
						label += " *"
					}
					if f.Button(remoteScope.At(i), label).Ghost().Small().Clicked() {
						m.selectedRemote = remote
					}
				}
			})
		}
		f.Divider()
		f.ScrollArea(logScrollID).Grow(1).Children(func() {
			for _, kind := range []int{0, 1, 2} {
				if kind == 2 {
					f.Text(text(i18n.Tags)).Muted().FontSize(12).PadY(7)
				} else if kind == 1 {
					f.Text(text(i18n.RemoteBranches)).Muted().FontSize(12).PadY(7)
				} else {
					f.Text(text(i18n.LocalBranches)).Muted().FontSize(12).PadY(7)
				}
				for i, ref := range m.refs {
					if kind == 0 && (ref.Remote || ref.Tag) || kind == 1 && (!ref.Remote || ref.Tag) || kind == 2 && !ref.Tag {
						continue
					}
					label := ref.Name
					if !ref.Remote && !ref.Tag && ref.Name == m.status.Branch {
						label = "* " + label
					}
					var row flex.Box
					f.Row().FillWidth().Height(30).PadX(8).Gap(8).Items(flex.Center).Children(func() {
						row = f.Row().Grow(1).FillHeight().Gap(18).Items(flex.Center).
							HitWith(branchScope.At(i), ref.Name, ui.HitOptions{Role: ui.RoleButton, Hint: ref.SHA}).Children(func() {
							f.Text(label).FontSize(12)
							f.Text(ref.SHA[:8]).Mono().Muted().FontSize(11)
							if ref.Track != "" {
								f.Text(ref.Track).Muted().FontSize(11)
							}
							if ref.Upstream != "" {
								f.Text(ref.Upstream).Muted().FontSize(11)
							}
						})
						if !ref.Remote && !ref.Tag && ref.Name != m.status.Branch {
							if f.Button(branchSwitchScope.At(i), text(i18n.SwitchBranch)).Ghost().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
								m.write("switch-branch", ref.Name, "")
							}
							if f.Button(tree.ID(feature, fmt.Sprintf("rename-branch-%d", i)), text(i18n.RenameBranch)).Ghost().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
								m.branchRenameOld = ref.Name
								m.localBranchName.SetText(ref.Name)
							}
							if f.Button(tree.ID(feature, fmt.Sprintf("delete-branch-%d", i)), text(i18n.DeleteBranch)).Ghost().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
								m.busy = true
								m.activeOperation = "delete-branch-plan"
								m.deletePlan = nil
								m.mailbox.PlanDeleteBranch(m.ctx, m.repository, ref.Name, m.refsRequest, m.invalidator.Invalidate)
							}
						}
						if !ref.Remote && !ref.Tag {
							upstream := ref.Upstream
							if upstream != "" {
								if f.Button(tree.ID(feature, fmt.Sprintf("unset-upstream-%d", i)), text(i18n.UnsetUpstream)).Ghost().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
									m.write("set-upstream", ref.Name, ref.Name+"\x00")
								}
							} else if remoteForBranch(m.refs, ref.Name) != "" {
								upstream = remoteForBranch(m.refs, ref.Name)
								if f.Button(tree.ID(feature, fmt.Sprintf("set-upstream-%d", i)), text(i18n.SetUpstream)).Ghost().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
									m.write("set-upstream", ref.Name, ref.Name+"\x00"+upstream)
								}
							}
						}
						if ref.Remote {
							localName := ref.Name
							if slash := strings.IndexByte(localName, '/'); slash >= 0 {
								localName = localName[slash+1:]
							}
							if f.Button(tree.ID(feature, fmt.Sprintf("checkout-remote-%d", i)), text(i18n.CheckoutRemoteBranch)).Ghost().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
								m.busy = true
								m.activeOperation = "remote-checkout-plan"
								m.operationError = ""
								m.mailbox.PlanRemoteCheckout(m.ctx, m.repository, ref.FullName, localName, m.refsRequest, m.invalidator.Invalidate)
							}
						}
					})
					if row.Clicked() {
						m.logFilter.Ref = ref.FullName
						m.resetLog()
						m.Active = keymap.ShowLog
						m.loadLog()
					}
				}
			}
			if m.branchRenameOld != "" {
				f.Row().FillWidth().Gap(8).Children(func() {
					if f.Field(&m.localBranchName, tree.ID(feature, "rename-branch-name"), text(i18n.LocalBranchName), text(i18n.LocalBranchName)).Grow(1).Submitted() {
						m.write("rename-branch", m.branchRenameOld, m.branchRenameOld+"\x00"+strings.TrimSpace(m.localBranchName.Text()))
					}
					if f.Button(tree.ID(feature, "rename-branch-submit"), text(i18n.RenameBranch)).Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
						m.write("rename-branch", m.branchRenameOld, m.branchRenameOld+"\x00"+strings.TrimSpace(m.localBranchName.Text()))
					}
				})
			}
		})
	})
}

func remoteForBranch(refs []git.Ref, local string) string {
	for _, ref := range refs {
		if ref.Remote && strings.HasSuffix(ref.FullName, "/"+local) {
			return ref.FullName
		}
	}
	return ""
}

func (m *Model) paintChangesSidebar(f *flex.Frame) {
	text := m.Language.Text
	f.Column().FillHeight().Background(ui.RGB(29, 31, 36)).Children(func() {
		f.Row().FillWidth().Height(38).PadX(8).Gap(4).Items(flex.Center).Children(func() {
			if f.Button(tree.ID(feature, "commit-tab"), text(i18n.CommitButton)).Ghost().Small().Clicked() {
				m.Active = keymap.ShowChanges
			}
			if f.Button(tree.ID(feature, "shelf-tab"), text(i18n.Stash)).Ghost().Small().Clicked() {
				m.Active = keymap.ShowStash
			}
			f.Spacer()
			if f.Button(refreshID, text(i18n.Refresh)).Ghost().Small().Disabled(m.busy).Clicked() {
				m.refresh()
			}
		})
		f.Row().FillWidth().Height(28).PadX(8).Gap(4).Children(func() {
			treeLabel := text(i18n.TreeView)
			if m.treeMode {
				treeLabel += " *"
			}
			if f.Button(treeModeID, treeLabel).Ghost().Small().Clicked() {
				m.treeMode = true
				m.saveRepositorySettings()
			}
			flatLabel := text(i18n.FlatView)
			if !m.treeMode {
				flatLabel += " *"
			}
			if f.Button(flatModeID, flatLabel).Ghost().Small().Clicked() {
				m.treeMode = false
				m.saveRepositorySettings()
			}
		})
		f.Divider()
		f.ScrollArea(fileListID).Grow(1).PadX(8).Children(func() {
			if m.treeMode {
				m.paintChangesTree(f)
			} else {
				m.paintGroup(f, i18n.ConflictedFiles, true, func(file git.File) bool { return file.Conflict })
				m.paintGroup(f, i18n.StagedFiles, true, func(file git.File) bool { return file.Index != '.' && !file.Untracked && !file.Conflict })
				m.paintGroup(f, i18n.UnstagedFiles, false, func(file git.File) bool { return file.Worktree != '.' && !file.Untracked && !file.Conflict })
				m.paintGroup(f, i18n.UntrackedFiles, false, func(file git.File) bool { return file.Untracked })
				m.paintIgnored(f)
			}
		})
		f.Divider()
		f.Column().FillWidth().Height(232).Pad(8).Gap(6).Children(func() {
			f.Textarea(m.message.snapshot(messageID, text(i18n.CommitMessage))).FillWidth().Height(68)
			if f.Checkbox(amendID, m.amend).Label(text(i18n.Amend)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
				m.amend = !m.amend
			}
			if len(m.messageHistory) > 0 {
				if index, ok := f.Select(tree.Sub(feature, tree.Name("commit-message-history-options")), &m.messageHistorySelect, m.messageLabels, m.selectedHistory).Placeholder(text(i18n.CommitMessageHistory)).Small().Picked(); ok {
					m.selectedHistory = index
					m.message.SetText(m.messageHistory[index])
				}
			}
			f.Row().FillWidth().Gap(4).Children(func() {
				if f.Button(commitID, text(i18n.CommitButton)).Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite || strings.TrimSpace(m.message.Text()) == "").Clicked() {
					kind := "commit"
					if m.amend {
						kind = "amend"
					}
					m.write(kind, "", m.message.Text())
				}
				if f.Button(commitPushID, text(i18n.CommitAndPush)).Ghost().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite || strings.TrimSpace(m.message.Text()) == "").Clicked() {
					m.commitAndPush = true
					kind := "commit"
					if m.amend {
						kind = "amend"
					}
					m.write(kind, "", m.message.Text())
				}
			})
		})
	})
}

func (m *Model) paintChangesDiff(f *flex.Frame) {
	text := m.Language.Text
	f.Column().Fill().Pad(12).Gap(8).Children(func() {
		if m.showErrorDetails && m.operationError != "" {
			f.ScrollArea(errorScrollID).Height(88).Children(func() {
				for line := range strings.SplitSeq(m.operationError, "\n") {
					f.Text(line).Mono().FontSize(11)
				}
			})
		}
		f.Row().FillWidth().Height(30).Gap(8).Items(flex.Center).Children(func() {
			f.Text(text(i18n.Changes)).Bold().FontSize(14)
			if f.Checkbox(ignoredToggleID, m.restoreIgnored).Label(text(i18n.ShowIgnored)).Small().Clicked() {
				m.restoreIgnored = !m.restoreIgnored
				if m.restoreIgnored && len(m.ignoredPaths) == 0 {
					m.loadIgnored()
				}
			}
			if m.selected >= 0 && m.selected < len(m.status.Files) {
				f.Text(m.status.Files[m.selected].Path).Muted().FontSize(12).Ellipsis()
			}
		})
		f.Divider()
		if m.selected < 0 || m.selected >= len(m.status.Files) {
			f.Spacer()
			f.Text(text(i18n.SelectFile)).Muted()
			f.Spacer()
			return
		}
		file := m.status.Files[m.selected]
		f.Row().FillWidth().Gap(8).Items(flex.Center).Children(func() {
			if file.Conflict {
				if f.Button(tree.ID(feature, "accept-current"), text(i18n.AcceptCurrent)).Small().Disabled(m.busy).Clicked() {
					m.resolveConflict(file.Path, "ours")
				}
				if f.Button(tree.ID(feature, "accept-incoming"), text(i18n.AcceptIncoming)).Small().Disabled(m.busy).Clicked() {
					m.resolveConflict(file.Path, "theirs")
				}
				if f.Button(tree.ID(feature, "accept-both"), text(i18n.AcceptBoth)).Small().Disabled(m.busy).Clicked() {
					m.resolveConflict(file.Path, "both")
				}
				return
			}
			if f.Button(stageID, text(i18n.StageFile)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite || (file.Worktree == '.' && !file.Untracked)).Clicked() {
				m.write("stage", file.Path, "")
			}
			if f.Button(unstageID, text(i18n.UnstageFile)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite || file.Index == '.' || file.Untracked).Clicked() {
				m.write("unstage", file.Path, "")
			}
			if file.Index != '.' && file.Worktree != '.' && !file.Untracked {
				label := text(i18n.StagedDiff)
				if m.selectedStaged {
					label = text(i18n.WorkingDiff)
				}
				if f.Button(toggleDiffID, label).Ghost().Small().Clicked() {
					m.selectedStaged = !m.selectedStaged
					m.diff, m.showComparison = "", false
					m.request++
					m.mailbox.Diff(m.ctx, m.repository, file, m.selectedStaged, m.request, m.invalidator.Invalidate)
				}
			}
		})
		if len(m.hunks) > 0 && m.selected >= 0 && m.selected < len(m.status.Files) {
			f.Row().FillWidth().Gap(4).Children(func() {
				for index := range m.hunks {
					label := text(i18n.StageHunk)
					if m.selectedStaged {
						label = text(i18n.UnstageHunk)
					}
					if m.selectedHunks[index] {
						label = "[x] " + label
					}
					if f.Button(hunkScope.At(index), fmt.Sprintf("%s %d", label, index+1)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
						m.selectedHunks[index] = !m.selectedHunks[index]
					}
				}
			})
			selected := make([]repo.HunkSelection, 0, len(m.selectedHunks))
			for index, checked := range m.selectedHunks {
				if checked {
					selected = append(selected, repo.HunkSelection{Index: index, Expected: fmt.Sprintf("%x", sha256.Sum256(m.hunks[index].Patch))})
				}
			}
			if len(selected) > 0 {
				f.Row().FillWidth().Gap(4).Children(func() {
					applyLabel := text(i18n.ApplySelectedHunks)
					if m.selectedStaged {
						applyLabel = text(i18n.UnstageHunk)
					}
					if f.Button(applyHunksID, applyLabel).Small().Disabled(m.busy).Clicked() {
						kind := "stage-hunk"
						if m.selectedStaged {
							kind = "unstage-hunk"
						}
						var packed strings.Builder
						for i, item := range selected {
							if i > 0 {
								packed.WriteByte(',')
							}
							fmt.Fprintf(&packed, "%d:%s", item.Index, item.Expected)
						}
						m.write(kind, file.Path, packed.String())
					}
					if !m.selectedStaged && f.Button(restoreHunksID, text(i18n.RestoreSelectedHunks)).Ghost().Small().Disabled(m.busy).Clicked() {
						m.request++
						m.busy = true
						m.activeOperation = "restore-hunks-plan"
						m.mailbox.PlanRestoreHunks(m.ctx, m.repository, file.Path, selected, m.request, m.invalidator.Invalidate)
					}
				})
			}
		}
		if file.Worktree != '.' || file.Untracked {
			if f.Button(discardID, text(i18n.DiscardFile)).Small().Destructive().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
				m.request++
				m.activeOperation = "discard-plan"
				m.busy = true
				m.mailbox.PlanDiscard(m.ctx, m.repository, file.Path, m.request, m.invalidator.Invalidate)
			}
		}
		if m.showComparison {
			m.diffSlot = f.Slot().Grow(1)
		} else {
			f.ScrollArea(diffScrollID).Grow(1).Children(func() {
				if m.diff == "" {
					f.Text(text(i18n.SelectFile)).Muted()
				} else {
					for line := range strings.SplitSeq(m.diff, "\n") {
						f.Text(line).Mono().FontSize(12)
					}
				}
			})
		}
	})
	if m.discardPlan != nil {
		dialog := f.Dialog(flex.DialogSpec{
			Title: text(i18n.ConfirmDiscard), Description: m.discardPlan.Path, Width: 420,
			Scope: tree.ID(feature, "discard-dialog"), DismissID: tree.ID(feature, "discard-dismiss"),
			CloseID: tree.ID(feature, "discard-close"), InitialFocusID: discardConfirmID,
		})
		dialog.Body(func() {
			f.Text(fmt.Sprintf("%s: %s", text(i18n.DiscardFile), m.discardPlan.Path)).FontSize(12)
		})
		dialog.Footer(func() {
			if f.Button(tree.ID(feature, "discard-cancel"), text(i18n.Cancel)).Outline().Clicked() || dialog.Dismissed() {
				m.discardPlan = nil
				m.busy = false
				m.activeOperation = ""
			}
			if f.Button(discardConfirmID, text(i18n.DiscardFile)).Destructive().Disabled(m.busy && m.activeOperation == "discard").Clicked() {
				plan := *m.discardPlan
				m.discardPlan = nil
				m.busy = true
				m.activeOperation = "discard"
				m.mailbox.Discard(m.ctx, m.repository, plan, m.invalidator.Invalidate)
			}
		})
	}
	if m.restorePlan != nil {
		dialog := f.Dialog(flex.DialogSpec{Title: text(i18n.ConfirmRestoreHunks), Description: m.restorePlan.Path, Width: 420,
			Scope: tree.ID(feature, "restore-hunks-dialog"), DismissID: tree.ID(feature, "restore-hunks-dismiss"), CloseID: tree.ID(feature, "restore-hunks-close"), InitialFocusID: restoreConfirmID})
		dialog.Body(func() {
			f.Text(fmt.Sprintf("%s: %d", m.restorePlan.Path, len(m.restorePlan.Selections))).FontSize(12)
		})
		dialog.Footer(func() {
			if f.Button(tree.ID(feature, "restore-hunks-cancel"), text(i18n.Cancel)).Outline().Clicked() || dialog.Dismissed() {
				m.restorePlan = nil
				m.busy = false
				m.activeOperation = ""
			}
			if f.Button(restoreConfirmID, text(i18n.RestoreSelectedHunks)).Destructive().Clicked() {
				plan := *m.restorePlan
				m.restorePlan = nil
				m.busy = true
				m.activeOperation = "restore-hunks"
				m.mailbox.RestoreHunks(m.ctx, m.repository, plan, m.invalidator.Invalidate)
			}
		})
	}
}

func (m *Model) resolveConflict(file, side string) {
	if m.repository == nil || m.busy {
		return
	}
	m.busy, m.activeOperation, m.operationError = true, "resolve-conflict", ""
	m.mailbox.ResolveConflict(m.ctx, m.repository, file, side, m.invalidator.Invalidate)
}

func (m *Model) paintStashes(f *flex.Frame) {
	text := m.Language.Text
	f.Column().Fill().Pad(12).Gap(8).Children(func() {
		f.Text(text(i18n.Stash)).Bold().FontSize(14)
		f.Row().FillWidth().Gap(6).Children(func() {
			if f.Field(&m.stashMessage, stashMessageID, text(i18n.StashMessage), "").Grow(1).Submitted() {
				m.stashWrite("save", "", m.stashMessage.Text())
			}
			if f.Button(stashSaveID, text(i18n.SaveStash)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
				m.stashWrite("save", "", m.stashMessage.Text())
			}
		})
		if f.Checkbox(stashUntrackedID, m.stashIncludeUntracked).Label(text(i18n.IncludeUntracked)).Small().Clicked() {
			m.stashIncludeUntracked = !m.stashIncludeUntracked
		}
		f.Divider()
		f.Row().Fill().Grow(1).Gap(10).Children(func() {
			f.ScrollArea(tree.ID(feature, "stash-list")).Width(310).FillHeight().Children(func() {
				for index, stash := range m.stashes {
					label := fmt.Sprintf("%s  %s", stash.Ref, stash.Subject)
					id := tree.Sub(feature, tree.Name("stash-rows")).At(index)
					if f.Button(id, label).Ghost().Small().Clicked() {
						m.selectStash(stash)
					}
				}
			})
			f.Divider()
			f.Column().Fill().Gap(6).Children(func() {
				selected := -1
				for index := range m.stashes {
					if m.stashes[index].SHA == m.selectedStash {
						selected = index
					}
				}
				if selected >= 0 {
					stash := m.stashes[selected]
					f.Text(strings.Join([]string{stash.Branch, stash.Date}, "  ")).Muted().FontSize(11)
					f.Row().FillWidth().Gap(5).Children(func() {
						for _, operation := range []struct {
							id   ui.ID
							key  i18n.Key
							name string
						}{{tree.ID(feature, "stash-apply"), i18n.ApplyStash, "apply"}, {tree.ID(feature, "stash-pop"), i18n.PopStash, "pop"}} {
							if f.Button(operation.id, text(operation.key)).Small().Disabled(m.busy).Clicked() {
								m.stashWrite(operation.name, stash.Ref, "")
							}
						}
						if f.Button(tree.ID(feature, "stash-branch"), text(i18n.BranchFromStash)).Small().Disabled(m.busy).Clicked() && strings.TrimSpace(m.stashBranchName.Text()) != "" {
							m.stashWrite("branch", stash.Ref, strings.TrimSpace(m.stashBranchName.Text()))
						}
						if m.stashPending == stash.Ref {
							if f.Button(stashDropConfirmID, text(i18n.ConfirmDiscard)).Destructive().Small().Clicked() {
								m.stashWrite("drop", stash.Ref, "")
							}
						} else if f.Button(tree.ID(feature, "stash-drop"), text(i18n.DropStash)).Destructive().Small().Clicked() {
							m.stashPending = stash.Ref
						}
					})
					f.Field(&m.stashBranchName, stashBranchNameID, text(i18n.LocalBranchName), "").FillWidth()
				}
				f.ScrollArea(tree.ID(feature, "stash-diff")).Grow(1).Children(func() {
					for _, line := range strings.Split(m.stashDiff, "\n") {
						f.Text(line).Mono().FontSize(11)
					}
				})
			})
		})
	})
}

func (m *Model) paintTags(f *flex.Frame) {
	text := m.Language.Text
	f.Column().Fill().Pad(12).Gap(8).Children(func() {
		f.Text(text(i18n.Tags)).Bold().FontSize(14)
		f.Row().FillWidth().Gap(6).Children(func() {
			f.Field(&m.tagName, tagNameID, text(i18n.TagName), "").Grow(1)
			f.Field(&m.tagTarget, tagTargetID, text(i18n.HistoryTarget), "HEAD").Grow(1)
		})
		f.Row().FillWidth().Gap(6).Children(func() {
			f.Field(&m.tagMessage, tagMessageID, text(i18n.TagMessage), "").Grow(1)
			if f.Button(tree.ID(feature, "tag-create"), text(i18n.CreateTag)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite || strings.TrimSpace(m.tagName.Text()) == "").Clicked() {
				m.tagWrite("create", strings.TrimSpace(m.tagName.Text()), strings.TrimSpace(m.tagTarget.Text()), m.tagMessage.Text())
			}
		})
		f.Divider()
		f.ScrollArea(tree.ID(feature, "tag-list")).Grow(1).Children(func() {
			for index, ref := range m.refs {
				if !ref.Tag {
					continue
				}
				f.Row().FillWidth().Height(28).Gap(6).Items(flex.Center).Children(func() {
					f.Text(ref.Name).Grow(1)
					f.Text(ref.SHA[:min(8, len(ref.SHA))]).Mono().Muted().FontSize(11)
					if m.tagPending == ref.Name {
						if f.Button(tree.ID(feature, "tag-delete-confirm"), text(i18n.ConfirmDeleteTag)).Destructive().Small().Clicked() {
							m.tagWrite("delete", ref.Name, "", "")
						}
					} else if f.Button(tree.Sub(feature, tree.Name("tag-delete")).At(index), text(i18n.DeleteBranch)).Ghost().Small().Clicked() {
						m.tagPending = ref.Name
					}
				})
			}
		})
	})
}

func (m *Model) tagWrite(operation, name, target, message string) {
	if m.repository == nil || m.busy {
		return
	}
	m.busy, m.activeOperation, m.operationError = true, "tag-"+operation, ""
	m.mailbox.TagWrite(m.ctx, m.repository, operation, name, target, message, m.invalidator.Invalidate)
}

func (m *Model) paintChangesTree(f *flex.Frame) {
	indexes := make([]int, 0, len(m.status.Files))
	for index := range m.status.Files {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool { return m.status.Files[indexes[i]].Path < m.status.Files[indexes[j]].Path })
	lastGroup := -1
	seenDirectories := make(map[string]struct{})
	if m.expandedPaths == nil {
		m.expandedPaths = make(map[string]bool)
	}
	for _, index := range indexes {
		file := m.status.Files[index]
		group := 0
		title := i18n.StagedFiles
		staged := true
		switch {
		case file.Conflict:
			group, title = -1, i18n.ConflictedFiles
		case file.Untracked:
			group, title, staged = 2, i18n.UntrackedFiles, false
		case file.Worktree != '.':
			group, title, staged = 1, i18n.UnstagedFiles, false
		}
		if group != lastGroup {
			f.Text(m.Language.Text(title)).Muted().FontSize(11).PadY(5)
			lastGroup = group
		}
		directory := path.Dir(file.Path)
		parts := strings.Split(directory, "/")
		hidden := false
		for level := range parts {
			prefix := strings.Join(parts[:level+1], "/")
			if prefix == "." {
				continue
			}
			if _, ok := seenDirectories[prefix]; ok {
				continue
			}
			seenDirectories[prefix] = struct{}{}
			label := path.Base(prefix)
			folderID := tree.ID(feature, "change-folder-"+prefix)
			if _, exists := m.expandedPaths[prefix]; !exists {
				m.expandedPaths[prefix] = true
			}
			if f.Button(folderID, label+"/").Ghost().Small().Clicked() {
				m.expandedPaths[prefix] = !m.expandedPaths[prefix]
			}
			if !m.expandedPaths[prefix] {
				hidden = true
				break
			}
		}
		if hidden {
			continue
		}
		depth := strings.Count(directory, "/") + 1
		if directory == "." {
			depth = 0
		}
		label := file.Path
		if file.Original != "" {
			label = file.Original + " -> " + file.Path
		}
		id := fileScope.At(index * 2)
		if staged {
			id = fileScope.At(index*2 + 1)
		}
		selected := m.selected == index && m.selectedStaged == staged
		color := ui.RGB(29, 31, 36)
		if selected {
			color = ui.RGB(43, 70, 117)
		}
		row := f.Row().FillWidth().Height(26).PadSides(0, 4, 0, float32(7+depth*12)).Gap(5).Items(flex.Center).
			Background(color).HoverBackground(ui.RGB(47, 51, 59)).Hit(id, label).Children(func() {
			f.Text(fileStatusLabel(file)).Mono().FontSize(10).Color(fileStatusColor(file))
			f.Text(label).FontSize(12).Ellipsis()
		})
		if row.Clicked() {
			m.selectFile(index, staged)
		}
	}
	m.paintIgnored(f)
}

func (m *Model) paintIgnored(f *flex.Frame) {
	if !m.restoreIgnored || len(m.ignoredPaths) == 0 {
		return
	}
	f.Text(m.Language.Text(i18n.ShowIgnored)).Muted().FontSize(11).PadY(5)
	for index, ignored := range m.ignoredPaths {
		f.Row().FillWidth().Height(24).PadX(7).Hit(tree.Sub(feature, tree.Name("ignored-paths")).At(index), ignored).Children(func() {
			f.Text(ignored).Muted().FontSize(11).Ellipsis()
		})
	}
}

func (m *Model) paintLog(f *flex.Frame) {
	text := m.Language.Text
	f.Row().Fill().Children(func() {
		branchWidth := min(max(m.logBranchesSplit.Pos, float32(110)), max(float32(110), m.width*0.18))
		showBranchSidebar := m.width >= 1000 && m.panelVisible("log-branches")
		if showBranchSidebar {
			f.Column().Width(branchWidth).FillHeight().Pad(7).Gap(5).Background(ui.RGB(29, 31, 36)).Children(func() {
				f.Text(text(i18n.Branches)).Bold().FontSize(11).Ellipsis()
				f.ScrollArea(tree.ID(feature, "log-branches")).Grow(1).Children(func() {
					localTitle, remoteTitle := false, false
					for i, ref := range m.refs {
						if ref.Tag {
							continue
						}
						if ref.Remote && !remoteTitle {
							f.Text(text(i18n.RemoteBranches)).Muted().FontSize(10).PadY(4)
							remoteTitle = true
						}
						if !ref.Remote && !localTitle {
							f.Text(text(i18n.LocalBranches)).Muted().FontSize(10).PadY(4)
							localTitle = true
						}
						label := ref.Name
						if !ref.Remote && ref.Name == m.status.Branch {
							label = "● " + label
						}
						background := ui.RGB(29, 31, 36)
						if m.logFilter.Ref == ref.FullName {
							background = ui.RGB(44, 70, 116)
						}
						row := f.Row().FillWidth().Height(25).PadX(5).Background(background).
							HoverBackground(ui.RGB(47, 51, 59)).HitWith(m.logBranchScope.At(i), ref.FullName,
							ui.HitOptions{Role: ui.RoleButton, Hint: ref.SHA}).Children(func() {
							f.Text(label).FontSize(11).Ellipsis()
						})
						if row.Clicked() {
							m.logRef.SetText(ref.FullName)
							m.applyLogFilter()
						}
					}
				})
			})
			f.Splitter(tree.ID(feature, "log-branches-split"), &m.logBranchesSplit).Mins(105, 260)
		}
		f.Column().Grow(1).FillHeight().Pad(4).Gap(4).Children(func() {
			listWidth := max(float32(250), m.width-branchWidth-350)
			label := text(i18n.Log)
			if m.logFilter.Ref != "" {
				label = m.logFilter.Ref
			}
			f.Row().FillWidth().Height(24).Items(flex.Center).Gap(5).Children(func() {
				f.Text(label).Bold().FontSize(14).PadY(2).Ellipsis()
				f.Spacer()
				if f.Checkbox(tree.ID(feature, "toggle-log-branches"), m.panelVisible("log-branches")).Label(text(i18n.Branches)).Small().Clicked() {
					m.setPanelVisible("log-branches", !m.panelVisible("log-branches"))
				}
				if f.Checkbox(tree.ID(feature, "toggle-log-details"), m.panelVisible("log-details")).Label(text(i18n.CommitFiles)).Small().Clicked() {
					m.setPanelVisible("log-details", !m.panelVisible("log-details"))
				}
			})
			if m.status.Operation.Kind != git.OperationNone {
				f.Row().FillWidth().Height(30).Gap(6).Items(flex.Center).Children(func() {
					f.Text(text(i18n.OperationInProgress)).Color(ui.RGB(222, 175, 94)).FontSize(12)
					if m.status.Operation.Target != "" {
						f.Text(m.status.Operation.Target[:min(8, len(m.status.Operation.Target))]).Mono().FontSize(11)
					}
					if m.status.Operation.Current > 0 && m.status.Operation.Remaining > 0 {
						f.Text(fmt.Sprintf("%s %d/%d", text(i18n.OperationProgress), m.status.Operation.Current, m.status.Operation.Current+m.status.Operation.Remaining)).Muted().FontSize(11)
					}
					conflictsRemain := slices.ContainsFunc(m.status.Files, func(file git.File) bool { return file.Conflict })
					if f.Button(continueOperationID, text(i18n.ContinueOperation)).Small().Disabled(m.busy || conflictsRemain || !m.Git.Version.Capabilities().CanWrite).Clicked() {
						m.resolveOperation("continue")
					}
					if m.status.Operation.Kind == git.OperationRebase || m.status.Operation.Kind == git.OperationCherryPick {
						if f.Button(skipOperationID, text(i18n.SkipOperation)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
							m.resolveOperation("skip")
						}
					}
					if f.Button(abortOperationID, text(i18n.AbortOperation)).Destructive().Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite).Clicked() {
						m.resolveOperation("abort")
					}
				})
			} else {
				if m.width < 1000 {
					if f.Button(tree.ID(feature, "history-tools-toggle"), text(i18n.HistoryTarget)).Small().Clicked() {
						m.historyToolsOpen = !m.historyToolsOpen
					}
				}
				if m.width >= 1000 || m.historyToolsOpen {
					f.Row().FillWidth().Height(26).Gap(4).Children(func() {
						placeholder := ""
						if m.historyAction == "cherry-pick" {
							placeholder = text(i18n.CherryPickTarget)
						}
						if f.Field(&m.historyTarget, historyTargetID, text(i18n.HistoryTarget), placeholder).Grow(1).Submitted() && m.historyAction != "" {
							m.planHistory(m.historyAction)
						}
					})
					f.Row().FillWidth().Height(26).Gap(4).Children(func() {
						actions := []string{text(i18n.MergeBranch), text(i18n.RebaseOnto), text(i18n.CherryPickCommit), text(i18n.RevertCommit), text(i18n.ResetSoft), text(i18n.ResetMixed), text(i18n.ResetHard)}
						if index, ok := f.Select(tree.Sub(feature, tree.Name("history-action")), &m.historyActionSelect, actions, -1).Placeholder(text(i18n.Search)).Small().Picked(); ok {
							m.historyAction = []string{"merge", "rebase", "cherry-pick", "revert", "reset-soft", "reset-mixed", "reset-hard"}[index]
						}
						if f.Button(tree.ID(feature, "history-preview"), text(i18n.Search)).Small().Disabled(m.busy || !m.Git.Version.Capabilities().CanWrite || m.historyAction == "" || strings.TrimSpace(m.historyTarget.Text()) == "").Clicked() {
							m.planHistory(m.historyAction)
						}
					})
				}
			}
			if m.historyPlan != nil {
				f.Column().FillWidth().Gap(4).Pad(7).Background(ui.RGB(45, 42, 36)).Children(func() {
					f.Text(fmt.Sprintf("%s: %s -> %s", text(i18n.ConfirmHistoryOperation), m.historyPlan.Action, m.historyPlan.Resolved[:min(8, len(m.historyPlan.Resolved))])).FontSize(12)
					if len(m.historyPlan.DirtyFiles) > 0 {
						f.Text(fmt.Sprintf("%s %d", text(i18n.CheckoutChangesWarning), len(m.historyPlan.DirtyFiles))).FontSize(11)
						for _, file := range m.historyPlan.DirtyFiles[:min(5, len(m.historyPlan.DirtyFiles))] {
							f.Text(file).Mono().FontSize(11).Ellipsis()
						}
						if len(m.historyPlan.DirtyFiles) > 5 {
							f.Text(fmt.Sprintf("+%d", len(m.historyPlan.DirtyFiles)-5)).Muted().FontSize(11)
						}
					}
					if m.historyPlan.Action == "reset-hard" && len(m.historyPlan.DataLossFiles) > 0 {
						f.Text(text(i18n.HistoryDataLoss)).Color(ui.RGB(224, 94, 86)).FontSize(11)
						for _, file := range m.historyPlan.DataLossFiles[:min(5, len(m.historyPlan.DataLossFiles))] {
							f.Text(file).Mono().FontSize(11).Ellipsis()
						}
						if len(m.historyPlan.DataLossFiles) > 5 {
							f.Text(fmt.Sprintf("+%d", len(m.historyPlan.DataLossFiles)-5)).Muted().FontSize(11)
						}
					}
					if len(m.historyPlan.Commits) > 0 {
						f.Text(fmt.Sprintf("%s %d", text(i18n.HistoryCommitsAffected), len(m.historyPlan.Commits))).Muted().FontSize(11)
						for _, commit := range m.historyPlan.Commits[:min(5, len(m.historyPlan.Commits))] {
							f.Text(fmt.Sprintf("%s %s", commit.SHA[:8], commit.Subject)).FontSize(11).Ellipsis()
						}
						if len(m.historyPlan.Commits) > 5 {
							f.Text(fmt.Sprintf("+%d", len(m.historyPlan.Commits)-5)).Muted().FontSize(11)
						}
					}
					f.Row().FillWidth().Gap(6).Children(func() {
						if f.Button(historyConfirmID, text(i18n.ConfirmHistoryOperation)).Small().Destructive().Disabled(m.busy).Clicked() {
							plan := *m.historyPlan
							m.historyPlan = nil
							m.busy = true
							m.activeOperation = "history-operation"
							m.mailbox.ExecuteHistory(m.ctx, m.repository, plan, m.invalidator.Invalidate)
						}
						if f.Button(tree.ID(feature, "history-cancel"), text(i18n.Cancel)).Ghost().Small().Clicked() {
							m.historyPlan = nil
						}
					})
				})
			}
			if m.operationConfirm != "" {
				key, action := i18n.ConfirmAbortOperation, "abort"
				if m.operationConfirm == "skip" {
					key, action = i18n.ConfirmSkipOperation, "skip"
				}
				f.Row().FillWidth().Gap(6).Items(flex.Center).Children(func() {
					f.Text(text(key)).FontSize(11)
					if f.Button(tree.ID(feature, "operation-confirm"), text(i18n.ConfirmHistoryOperation)).Destructive().Small().Clicked() {
						m.executeOperation(action)
					}
					if f.Button(tree.ID(feature, "operation-cancel"), text(i18n.Cancel)).Ghost().Small().Clicked() {
						m.operationConfirm = ""
					}
				})
			}
			f.Row().FillWidth().Height(29).Gap(4).Children(func() {
				if f.Field(&m.logQuery, logFilterID, text(i18n.MessageFilter), text(i18n.MessageFilter)).Grow(1).Submitted() {
					m.applyLogFilter()
				}
				if f.Button(filterSubmitID, text(i18n.Search)).Clicked() {
					m.applyLogFilter()
				}
			})
			f.Row().FillWidth().Height(26).Gap(4).Children(func() {
				if f.Field(&m.logAuthor, logAuthorID, text(i18n.AuthorFilter), text(i18n.AuthorFilter)).Grow(1).Submitted() {
					m.applyLogFilter()
				}
				if f.Field(&m.logPath, logPathID, text(i18n.PathFilter), text(i18n.PathFilter)).Grow(1).Submitted() {
					m.applyLogFilter()
				}
			})
			f.Row().FillWidth().Height(26).Gap(4).Children(func() {
				if f.Field(&m.logSince, logSinceID, text(i18n.SinceDate), text(i18n.SinceDate)).Grow(1).Submitted() {
					m.applyLogFilter()
				}
				if f.Field(&m.logUntil, logUntilID, text(i18n.UntilDate), text(i18n.UntilDate)).Grow(1).Submitted() {
					m.applyLogFilter()
				}
			})
			f.Row().FillWidth().Height(26).Gap(4).Children(func() {
				if f.Field(&m.logRef, logRefID, text(i18n.RefFilter), text(i18n.RefFilter)).Grow(1).Submitted() {
					m.applyLogFilter()
				}
			})
			if m.logFilter != (repo.LogFilter{}) {
				if f.Button(filterClearID, text(i18n.ClearFilter)).Ghost().Clicked() {
					m.logQuery.SetText("")
					m.logAuthor.SetText("")
					m.logPath.SetText("")
					m.logSince.SetText("")
					m.logUntil.SetText("")
					m.logRef.SetText("")
					m.logFilter.Ref = ""
					m.applyLogFilter()
				}
			}
			f.Column().Grow(1).Children(func() {
				if m.logLoading {
					f.Text(text(i18n.Working)).Muted()
				}
				if len(m.commits) == 0 && !m.logLoading {
					f.Text(text(i18n.NoCommits)).Muted()
				}
				if len(m.commits) > 0 {
					columns := compactCommitColumns[:]
					graphWidth := m.graphWidth
					if graphWidth == 0 {
						graphWidth = 84
					}
					if listWidth >= graphWidth+400 {
						m.logColumns = [6]components.TableColumn{
							{Width: graphWidth}, {Width: max(listWidth-16-graphWidth-82-82-78-72, 120)},
							{Width: 82}, {Width: 82}, {Width: 78}, {Width: 72},
						}
						columns = m.logColumns[:]
					}
					m.logTableBounds = f.Table(&m.logTable, flex.TableSpec{
						Rows: logScope, Header: logHeaderScope, Columns: columns, Source: &m.logSource,
						Headers:   []string{"", "", text(i18n.AuthorColumn), text(i18n.CommitterColumn), "", ""},
						RowHeight: 25,
					}).FillWidth().Height(max(float32(55), m.height-310))
					if row, ok := m.logTableBounds.RowClicked(); ok && row < len(m.commits) {
						m.selectCommit(m.commits[row].SHA)
					}
				}
				if m.logHasMore {
					if f.Button(loadMoreID, text(i18n.LoadMore)).Disabled(m.logLoading || m.repository == nil).Clicked() {
						m.loadLog()
					}
				}
			})
		})
		if m.width >= 1000 {
			f.Splitter(tree.ID(feature, "log-details-split"), &m.logDetailsSplit).Mins(230, 250)
		}
		detailWidth := min(max(m.logDetailsSplit.Pos, float32(250)), max(float32(250), m.width*0.34))
		details := f.Column().FillHeight().Pad(8).Gap(6).Background(ui.RGB(29, 31, 36))
		if m.width >= 1000 {
			details.Width(detailWidth)
		} else {
			details.Width(min(float32(240), max(float32(190), (m.width-300)*0.42)))
		}
		details.Children(func() {
			if m.selectedCommit == "" {
				f.Text(text(i18n.NoCommits)).Muted().FontSize(12)
				return
			}
			f.Row().FillWidth().Items(flex.Center).Children(func() {
				f.Text(m.selectedCommit[:8]).Mono().FontSize(12)
				f.Spacer()
				f.Button(copySHAID, text(i18n.CopySHA)).Ghost().Small()
			})
			f.Text(text(i18n.CommitFiles)).Bold().FontSize(12).PadY(5)
			f.ScrollArea(logFilesID).Grow(1).Children(func() {
				for i, path := range m.commitFiles {
					selected := m.selectedCommitFile == path
					color := ui.RGB(29, 31, 36)
					if selected {
						color = ui.RGB(43, 70, 117)
					}
					row := f.Row().FillWidth().Height(25).PadX(6).Background(color).
						HoverBackground(ui.RGB(47, 51, 59)).Hit(commitFileScope.At(i), path).Children(func() {
						f.Text(path).FontSize(12).Ellipsis()
					})
					if row.Clicked() {
						m.selectCommitFile(path)
					}
				}
			})
			f.Divider()
			if m.selectedCommitFile == "" {
				f.Spacer()
			} else {
				f.Text(m.selectedCommitFile).Mono().FontSize(11).Ellipsis()
				if !m.commitFileReady {
					f.Text(text(i18n.Working)).Muted()
					f.Spacer()
				} else if m.commitFileDiff.Binary {
					f.ScrollArea(commitDiffScrollID).Grow(1).Children(func() {
						for line := range strings.SplitSeq(m.commitFileDiff.Patch, "\n") {
							f.Text(line).Mono().FontSize(12)
						}
					})
				} else {
					m.commitDiffSlot = f.Slot().Grow(1)
				}
			}
			if m.panelVisible("log-details") {
				f.Divider()
				f.ScrollArea(logDetailsID).Height(112).Children(func() {
					for line := range strings.SplitSeq(m.commitMessage, "\n") {
						f.Text(line).FontSize(12)
					}
				})
			}
		})
	})
}

func contains(r ui.Rect, x, y float32) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

func (m *Model) paintRepository(f *flex.Frame) {
	text := m.Language.Text
	f.Column().Fill().Pad(16).Gap(8).Children(func() {
		f.Text(text(i18n.Repositories)).Bold().FontSize(15)
		if m.showErrorDetails && m.operationError != "" {
			f.ScrollArea(errorScrollID).Height(88).Children(func() {
				for line := range strings.SplitSeq(m.operationError, "\n") {
					f.Text(line).Mono().FontSize(11)
				}
			})
		}
		f.Row().FillWidth().Gap(8).Items(flex.Center).Children(func() {
			if f.Field(&m.path, pathID, text(i18n.OpenPath), text(i18n.OpenPath)).Grow(1).Submitted() {
				m.Open(m.path.Text())
			}
			f.Button(browseID, text(i18n.Browse)).Disabled(m.busy)
			if f.Button(refreshID, text(i18n.Refresh)).Disabled(m.repository == nil || m.busy).Clicked() {
				m.refresh()
			}
		})
		if m.repository == nil {
			f.Text(text(i18n.SelectRepository)).Muted()
			return
		}
		f.Text(m.status.Root).Mono().FontSize(12)
	})
}

func (m *Model) containsDiff(x, y float32) bool {
	return contains(m.diffSlot.Rect(), x, y)
}

func (m *Model) paintGroup(f *flex.Frame, title i18n.Key, staged bool, include func(git.File) bool) {
	count := 0
	for _, file := range m.status.Files {
		if include(file) {
			count++
		}
	}
	if count == 0 {
		return
	}
	f.Text(fmt.Sprintf("%s  %d", m.Language.Text(title), count)).Muted().FontSize(11).PadY(6)
	for i, file := range m.status.Files {
		if !include(file) {
			continue
		}
		label := file.Path
		if file.Original != "" {
			label = file.Original + " -> " + file.Path
		}
		id := fileScope.At(i * 2)
		if staged {
			id = fileScope.At(i*2 + 1)
		}
		selected := m.selected == i && m.selectedStaged == staged
		color := ui.RGB(29, 31, 36)
		if selected {
			color = ui.RGB(43, 70, 117)
		}
		row := f.Row().FillWidth().Height(26).PadX(7).Gap(5).Items(flex.Center).Radius(3).
			Background(color).HoverBackground(ui.RGB(47, 51, 59)).Hit(id, label).Children(func() {
			f.Text(fileStatusLabel(file)).Mono().FontSize(10).Color(fileStatusColor(file))
			f.Text(label).FontSize(12).Ellipsis()
		})
		if row.Clicked() {
			m.selectFile(i, staged)
		}
	}
}

func fileStatusLabel(file git.File) string {
	if file.Conflict {
		return "U"
	}
	if file.Untracked {
		return "?"
	}
	if file.Index == 'R' || file.Worktree == 'R' {
		return "R"
	}
	if file.Index == 'D' || file.Worktree == 'D' {
		return "D"
	}
	if file.Index == 'A' || file.Worktree == 'A' {
		return "A"
	}
	if file.Index == 'M' || file.Worktree == 'M' {
		return "M"
	}
	return " "
}

func fileStatusColor(file git.File) ui.Color {
	switch fileStatusLabel(file) {
	case "U":
		return ui.RGB(224, 94, 86)
	case "?", "A":
		return ui.RGB(112, 190, 130)
	case "D":
		return ui.RGB(224, 94, 86)
	case "M", "R":
		return ui.RGB(222, 175, 94)
	default:
		return ui.RGB(150, 155, 164)
	}
}

func (m *Model) paintStatus(f *flex.Frame) {
	text := m.Language.Text
	f.Row().FillWidth().Height(36).PadX(16).Gap(12).Items(flex.Center).Children(func() {
		label := text(i18n.NoRepository)
		if m.repository != nil {
			label = m.status.Root
		}
		f.Text(label).Muted().FontSize(12)
		f.Spacer()
		if m.operationError != "" {
			f.Text(text(i18n.OperationFailed)).Color(ui.RGBA(160, 49, 46, 1)).FontSize(12)
			if f.Button(errorDetailsID, text(i18n.ErrorDetails)).Ghost().Clicked() {
				m.showErrorDetails = !m.showErrorDetails
			}
			if m.showErrorDetails {
				f.Text(m.operationError).Muted().Mono().FontSize(11).Ellipsis()
			}
		} else if m.busy {
			label := text(i18n.Working)
			if m.remoteProgress != "" {
				label = m.remoteProgress
			}
			f.Text(label).Muted().FontSize(12)
		}
		switch {
		case m.Checking:
			f.Text(text(i18n.CheckingGit)).Muted().FontSize(12)
		case m.DetectionErr != nil:
			f.Text(text(i18n.GitUnavailable)).Color(ui.RGBA(160, 49, 46, 1)).FontSize(12)
		case !m.Git.Version.Capabilities().CanWrite:
			f.Text(text(i18n.GitUnsupported)).Color(ui.RGBA(160, 49, 46, 1)).FontSize(12)
		default:
			f.Text(text(i18n.Ready)).Muted().FontSize(12)
		}
		if m.Git.Path != "" {
			f.Text(fmt.Sprintf("%s %s", text(i18n.GitVersion), m.Git.Version.Raw)).Mono().FontSize(12)
		}
	})
}
