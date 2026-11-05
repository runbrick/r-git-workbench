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
)

var stringsByLanguage = [...][ChangeGit + 1]string{
	English: {
		"Git Workbench", "Repository", "No repository", "Open repository", "Branch", "No branch",
		"Fetch", "Pull", "Push", "Search", "Changes", "Log", "Branches", "Stash", "Tags",
		"Repositories", "Select a repository", "Git unavailable", "Git version is below 2.34",
		"Git", "Checking Git", "Ready", "Find Git executable",
	},
	Chinese: {
		"Git 工作台", "仓库", "未打开仓库", "打开仓库", "分支", "无分支",
		"获取", "拉取", "推送", "搜索", "更改", "日志", "分支", "贮藏", "标签",
		"仓库列表", "选择一个仓库", "Git 不可用", "Git 版本低于 2.34",
		"Git", "正在检测 Git", "就绪", "查找 Git 可执行文件",
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
	if language > Chinese || key > ChangeGit {
		return ""
	}
	return stringsByLanguage[language][key]
}
