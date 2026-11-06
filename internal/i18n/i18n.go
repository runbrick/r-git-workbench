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
)

var stringsByLanguage = [...][PushTargets + 1]string{
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
	},
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
	if language > Chinese || key > PushTargets {
		return ""
	}
	return stringsByLanguage[language][key]
}
