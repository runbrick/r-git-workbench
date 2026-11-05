package shell

import (
	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/components"
	vivatext "github.com/vivaui/viva/text"
	"github.com/vivaui/viva/ui"
)

type commitDraft struct {
	editor *vivatext.Editor
	input  app.TextInput
	layout components.TextareaLayout
}

func (draft *commitDraft) ensure() *vivatext.Editor {
	if draft.editor == nil {
		draft.editor = vivatext.NewEditor("")
		draft.input.Editor = draft.editor
	}
	return draft.editor
}

func (draft *commitDraft) Text() string { return draft.ensure().String() }

func (draft *commitDraft) SetText(value string) {
	editor := draft.ensure()
	editor.SetSelection(vivatext.Range{Start: 0, End: editor.RuneCount()})
	editor.Insert(value)
}

func (draft *commitDraft) snapshot(id ui.ID, label string) components.TextareaSnapshot {
	editor := draft.ensure()
	value := editor.String()
	selection := editor.Selection()
	return components.TextareaSnapshot{
		ID: id, Label: label, Placeholder: label, Value: value,
		HideEditorText: true, ShowCaret: selection.Start == selection.End,
		Caret:          draft.byteOffset(value, editor.Caret()),
		SelectionStart: draft.byteOffset(value, selection.Start),
		SelectionEnd:   draft.byteOffset(value, selection.End), Layout: &draft.layout,
	}
}

func (draft *commitDraft) byteOffset(value string, runes int) int {
	for offset := range value {
		if runes == 0 {
			return offset
		}
		runes--
	}
	return len(value)
}

func (draft *commitDraft) move(event ui.KeyEvent) ui.EventResult {
	movement, extend := components.TextareaMovementForKey(event.Key, event.Modifiers)
	if movement == components.TextareaMoveNone {
		return ui.EventResult{}
	}
	editor := draft.ensure()
	value := editor.String()
	to := draft.layout.Move(value, draft.byteOffset(value, editor.Caret()), movement)
	editor.MoveTo(len([]rune(value[:to])), extend)
	return ui.EventResult{Handled: true, Repaint: true}
}
