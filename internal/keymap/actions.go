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
	ShowSettings
)

type Action struct {
	ID             ActionID
	Name           i18n.Key
	Key            string
	Context        Context
	NeedsSelection bool
}

type Context uint8

const (
	Global Context = iota
	Changes
	Log
	Branches
	Stash
	TextInput
	Dialog
)

type Binding struct {
	Action  string  `json:"action"`
	Chord   string  `json:"chord"`
	Context Context `json:"context"`
}

type Preset string

const (
	IntelliJ Preset = "intellij"
	VSCode   Preset = "vscode"
	Default  Preset = "default"
)

func Presets() map[Preset][]Binding {
	makeBindings := func(changes, log, branches, stash string) []Binding {
		return []Binding{
			{Action: "view.changes", Chord: changes, Context: Global},
			{Action: "view.log", Chord: log, Context: Global},
			{Action: "view.branches", Chord: branches, Context: Global},
			{Action: "view.stash", Chord: stash, Context: Global},
			{Action: "view.refresh", Chord: "Primary+F5", Context: Global},
			{Action: "commit.create", Chord: "Primary+K", Context: Changes},
			{Action: "log.search", Chord: "Primary+F", Context: Log},
			{Action: "diff.open", Chord: "Primary+D", Context: Changes},
		}
	}
	return map[Preset][]Binding{
		IntelliJ: makeBindings("Alt+0", "Alt+9", "Ctrl+Shift+`", "Alt+4"),
		VSCode:   makeBindings("Ctrl+Shift+G", "Ctrl+Shift+L", "Ctrl+Shift+B", "Ctrl+Shift+S"),
		Default:  makeBindings("Primary+1", "Primary+2", "Primary+3", "Primary+4"),
	}
}

func Conflicts(bindings []Binding) [][2]Binding {
	var conflicts [][2]Binding
	for i, left := range bindings {
		for _, right := range bindings[i+1:] {
			if left.Chord == right.Chord && left.Context == right.Context {
				conflicts = append(conflicts, [2]Binding{left, right})
			}
		}
	}
	return conflicts
}

var Actions = [...]Action{
	{ID: ShowChanges, Name: i18n.Changes, Key: "view.changes", Context: Global},
	{ID: ShowLog, Name: i18n.Log, Key: "view.log", Context: Global},
	{ID: ShowBranches, Name: i18n.Branches, Key: "view.branches", Context: Global},
	{ID: ShowStash, Name: i18n.Stash, Key: "view.stash", Context: Global},
	{ID: ShowTags, Name: i18n.Tags, Key: "view.tags", Context: Global},
	{ID: ShowRepositories, Name: i18n.Repositories, Key: "view.repositories", Context: Global},
}
