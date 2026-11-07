package i18n

import "strings"

type Language uint8

const (
	English Language = iota
	Chinese
)

type Key uint8

const (
	AppName Key = iota
	Repository
	NoRepository
	OpenRepository
	Branch
	NoBranch
	Fetch
	Pull
	Push
	Search
	Changes
	Log
	Branches
	Stash
	Tags
	Repositories
	SelectRepository
	GitUnavailable
	GitUnsupported
	GitVersion
	CheckingGit
	Ready
	ChangeGit
	OpenPath
	Refresh
	StagedFiles
	UnstagedFiles
	UntrackedFiles
	ConflictedFiles
	StageFile
	UnstageFile
	CommitMessage
	CommitButton
	WorkingDiff
	StagedDiff
	SelectFile
	Working
	OperationFailed
	Browse
	ErrorDetails
	LoadMore
	NoCommits
	CommitFiles
	CopySHA
	MessageFilter
	ClearFilter
	AuthorFilter
	PathFilter
	LocalBranches
	RemoteBranches
	NewBranch
	BranchName
	SwitchBranch
	RenameBranch
	DeleteBranch
	SetUpstream
	UnsetUpstream
	CheckoutRemoteBranch
	LocalBranchName
	ConfirmDeleteBranch
	Cancel
	CheckoutChangesWarning
	StashAndCheckout
	ForceCheckout
	ConfirmForceCheckout
	ForcePushWithLease
	ConfirmForcePush
	PushTargets
	PullMerge
	PullRebase
	PullFFOnly
	PullPolicyLabel
	SinceDate
	UntilDate
	AheadLabel
	BehindLabel
	PushPreview
	RefFilter
	RemoteTarget
	UpstreamDefault
	StageHunk
	UnstageHunk
	Amend
	CommitAndPush
	CommitMessageHistory
	DiscardFile
	ConfirmDiscard
	TreeView
	FlatView
	ApplySelectedHunks
	RestoreSelectedHunks
	ConfirmRestoreHunks
	ShowIgnored
	AuthorColumn
	CommitterColumn
	ResetSoft
	ResetMixed
	ResetHard
	HistoryTarget
	MergeBranch
	RebaseOnto
	ConfirmHistoryOperation
	HistoryCommitsAffected
	OperationInProgress
	ContinueOperation
	AbortOperation
	CherryPickCommit
	RevertCommit
	SkipOperation
	ConfirmAbortOperation
	ConfirmSkipOperation
	HistoryDataLoss
	OperationProgress
	CherryPickTarget
	SaveStash
	StashMessage
	IncludeUntracked
	ApplyStash
	PopStash
	DropStash
	BranchFromStash
	AcceptCurrent
	AcceptIncoming
	AcceptBoth
	CurrentVersion
	IncomingVersion
	StashSaved
	Settings
	LanguageSetting
	PullDefault
	Theme
	SaveSettings
	TagName
	TagMessage
	TagTarget
	CreateTag
	ConfirmDeleteTag
)

var stringsByLanguage = [...][ConfirmDeleteTag + 1]string{
	English: {
		"Git Workbench", "Repository", "No repository", "Open repository", "Branch", "No branch",
		"Fetch", "Pull", "Push", "Search", "Changes", "Log", "Branches", "Stash", "Tags",
		"Repositories", "Select a repository", "Git unavailable", "Git version is below 2.34",
		"Git", "Checking Git", "Ready", "Find Git executable",
		"Repository path", "Refresh", "Staged changes", "Unstaged changes", "Untracked files", "Conflicts",
		"Stage", "Unstage", "Commit message", "Commit", "Working tree diff", "Staged diff", "Select a file", "Working...", "Operation failed",
		"Browse...",
		"Error details",
		"Load more", "No commits", "Files in commit",
		"Copy SHA",
		"Search messages", "Clear filter",
		"Author", "Path",
		"Local branches", "Remote branches",
		"Create branch", "Branch name", "Switch",
		"Rename", "Delete", "Set upstream", "Unset upstream", "Checkout as local", "Local branch name",
		"Delete this branch?", "Cancel",
		"Uncommitted files may be affected:", "Stash and checkout", "Force checkout", "Confirm force checkout",
		"Force push with lease", "Confirm force push", "Push targets (local -> expected remote)",
		"Merge", "Rebase", "Fast-forward only",
		"Policy:",
		"Since (YYYY-MM-DD)", "Until (YYYY-MM-DD)",
		"ahead", "behind",
		"Push preview:",
		"Branch or ref",
		"Remote:",
		"Upstream",
		"Stage hunk", "Unstage hunk",
		"Amend", "Commit + Push", "Recent commit messages",
		"Discard", "Discard these working tree changes?", "Tree", "Flat",
		"Apply selected hunks", "Restore selected hunks", "Restore selected working tree changes?", "Show ignored files",
		"Author", "Committer",
		"Reset soft", "Reset mixed", "Reset hard", "Target commit or ref", "Merge branch", "Rebase onto",
		"Confirm Git operation", "Commits affected:", "Git operation in progress", "Continue", "Abort",
		"Cherry-pick", "Revert", "Skip", "Abort this Git operation and discard its in-progress changes?", "Skip the current commit and continue?",
		"Hard reset will discard:",
		"Progress",
		"Commit SHAs or refs, in order (comma or space separated)",
	},
	Chinese: {
		"Git 工作台", "仓库", "未打开仓库", "打开仓库", "分支", "无分支",
		"获取", "拉取", "推送", "搜索", "更改", "日志", "分支", "贮藏", "标签",
		"仓库列表", "选择一个仓库", "Git 不可用", "Git 版本低于 2.34",
		"Git", "正在检测 Git", "就绪", "查找 Git 可执行文件",
		"仓库路径", "刷新", "已暂存更改", "未暂存更改", "未跟踪文件", "冲突",
		"暂存", "取消暂存", "提交消息", "提交", "工作区差异", "暂存区差异", "选择文件", "正在执行...", "操作失败",
		"浏览...",
		"错误详情",
		"加载更多", "暂无提交", "提交文件",
		"复制 SHA",
		"搜索提交消息", "清除筛选",
		"作者", "路径",
		"本地分支", "远程分支",
		"新建分支", "分支名称", "切换",
		"重命名", "删除", "设置上游", "取消上游", "检出为本地分支", "本地分支名",
		"确认删除此分支？", "取消",
		"以下未提交文件可能受影响：", "暂存改动并检出", "强制检出", "确认强制检出",
		"带保护的强制推送", "确认强制推送", "推送目标（本地 -> 预期远端）",
		"合并", "变基", "仅快进",
		"策略：",
		"起始日期（YYYY-MM-DD）", "结束日期（YYYY-MM-DD）",
		"领先", "落后",
		"推送预览：",
		"分支或引用",
		"远端：",
		"上游",
		"暂存区块", "取消暂存区块",
		"修改上次提交", "提交 + 推送", "最近提交消息",
		"丢弃", "确认丢弃这些工作区更改？", "目录树", "扁平",
		"应用所选区块", "还原所选区块", "确认还原所选工作区更改？", "显示忽略文件",
		"作者", "提交者",
		"软重置", "混合重置", "硬重置", "目标提交或引用", "合并分支", "变基到",
		"确认 Git 操作", "受影响的提交：", "Git 操作进行中", "继续", "中止",
	},
}

func init() {
	stringsByLanguage[English][SaveStash] = "Save stash"
	stringsByLanguage[English][StashMessage] = "Stash message"
	stringsByLanguage[English][IncludeUntracked] = "Include untracked files"
	stringsByLanguage[English][ApplyStash] = "Apply"
	stringsByLanguage[English][PopStash] = "Pop"
	stringsByLanguage[English][DropStash] = "Drop"
	stringsByLanguage[English][BranchFromStash] = "Branch"
	stringsByLanguage[English][AcceptCurrent] = "Accept Current"
	stringsByLanguage[English][AcceptIncoming] = "Accept Incoming"
	stringsByLanguage[English][AcceptBoth] = "Accept Both"
	stringsByLanguage[English][CurrentVersion] = "Current version"
	stringsByLanguage[English][IncomingVersion] = "Incoming version"
	stringsByLanguage[English][StashSaved] = "Saved changes"
	stringsByLanguage[English][Settings] = "Settings"
	stringsByLanguage[English][LanguageSetting] = "Language"
	stringsByLanguage[English][PullDefault] = "Default pull policy"
	stringsByLanguage[English][Theme] = "Theme"
	stringsByLanguage[English][SaveSettings] = "Save settings"
	stringsByLanguage[English][TagName] = "Tag name"
	stringsByLanguage[English][TagMessage] = "Annotation message (optional)"
	stringsByLanguage[English][TagTarget] = "Target commit"
	stringsByLanguage[English][CreateTag] = "Create tag"
	stringsByLanguage[English][ConfirmDeleteTag] = "Confirm delete tag"
	stringsByLanguage[Chinese][SaveStash] = "保存贮藏"
	stringsByLanguage[Chinese][StashMessage] = "贮藏说明"
	stringsByLanguage[Chinese][IncludeUntracked] = "包含未跟踪文件"
	stringsByLanguage[Chinese][ApplyStash] = "应用"
	stringsByLanguage[Chinese][PopStash] = "弹出"
	stringsByLanguage[Chinese][DropStash] = "删除"
	stringsByLanguage[Chinese][BranchFromStash] = "创建分支"
	stringsByLanguage[Chinese][AcceptCurrent] = "接受当前版本"
	stringsByLanguage[Chinese][AcceptIncoming] = "接受传入版本"
	stringsByLanguage[Chinese][AcceptBoth] = "接受双方版本"
	stringsByLanguage[Chinese][CurrentVersion] = "当前版本"
	stringsByLanguage[Chinese][IncomingVersion] = "传入版本"
	stringsByLanguage[Chinese][StashSaved] = "已保存更改"
	stringsByLanguage[Chinese][Settings] = "设置"
	stringsByLanguage[Chinese][LanguageSetting] = "界面语言"
	stringsByLanguage[Chinese][PullDefault] = "默认拉取策略"
	stringsByLanguage[Chinese][Theme] = "主题"
	stringsByLanguage[Chinese][SaveSettings] = "保存设置"
	stringsByLanguage[Chinese][TagName] = "标签名称"
	stringsByLanguage[Chinese][TagMessage] = "注释（可选）"
	stringsByLanguage[Chinese][TagTarget] = "目标提交"
	stringsByLanguage[Chinese][CreateTag] = "创建标签"
	stringsByLanguage[Chinese][ConfirmDeleteTag] = "确认删除标签"
	stringsByLanguage[Chinese][HistoryDataLoss] = "硬重置将丢弃："
	stringsByLanguage[Chinese][OperationProgress] = "进度"
	stringsByLanguage[Chinese][CherryPickTarget] = "提交 SHA 或引用，按拣选顺序输入（逗号或空格分隔）"
	stringsByLanguage[Chinese][CherryPickCommit] = "拣选提交"
	stringsByLanguage[Chinese][RevertCommit] = "回退提交"
	stringsByLanguage[Chinese][SkipOperation] = "跳过"
	stringsByLanguage[Chinese][ConfirmAbortOperation] = "中止当前 Git 操作并丢弃其进行中的更改？"
	stringsByLanguage[Chinese][ConfirmSkipOperation] = "跳过当前提交并继续？"
}

func FromPreferred(languages []string) Language {
	for _, language := range languages {
		if strings.HasPrefix(strings.ToLower(language), "zh") {
			return Chinese
		}
		if strings.HasPrefix(strings.ToLower(language), "en") {
			return English
		}
	}
	return English
}

func (language Language) Text(key Key) string {
	if language > Chinese || key > CherryPickTarget {
		return ""
	}
	return stringsByLanguage[language][key]
}
