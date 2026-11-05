package keymap

import "r-git-workbench/internal/i18n"

type ActionID uint8

const (
	ShowChanges ActionID = iota
	ShowLog
	ShowBranches
	ShowStash
	ShowTags
	ShowRepositories
)

type Action struct {
	ID   ActionID
	Name i18n.Key
}

var Actions = [...]Action{
	{ShowChanges, i18n.Changes},
	{ShowLog, i18n.Log},
	{ShowBranches, i18n.Branches},
	{ShowStash, i18n.Stash},
	{ShowTags, i18n.Tags},
	{ShowRepositories, i18n.Repositories},
}
