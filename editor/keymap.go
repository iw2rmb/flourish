package editor

import "charm.land/bubbles/v2/key"

// KeyMap defines the editor key bindings.
//
// Bindings must be portable across terminals (ctrl/alt fallbacks).
type KeyMap struct {
	Left, Right, Up, Down                     key.Binding
	ParagraphUp, ParagraphDown                key.Binding
	PageUp, PageDown                          key.Binding
	ShiftLeft, ShiftRight, ShiftUp, ShiftDown key.Binding
	ParagraphShiftUp, ParagraphShiftDown      key.Binding
	WordLeft, WordRight                       key.Binding
	WordShiftLeft, WordShiftRight             key.Binding
	Home, End                                 key.Binding

	Backspace, Delete                 key.Binding
	DeleteWordBackward, KillLineRight key.Binding
	Enter                             key.Binding

	Undo, Redo key.Binding
}

// bindings returns all key bindings as a slice.
func (km KeyMap) bindings() []key.Binding {
	return []key.Binding{
		km.Left, km.Right, km.Up, km.Down,
		km.ParagraphUp, km.ParagraphDown,
		km.PageUp, km.PageDown,
		km.ShiftLeft, km.ShiftRight, km.ShiftUp, km.ShiftDown,
		km.ParagraphShiftUp, km.ParagraphShiftDown,
		km.WordLeft, km.WordRight, km.WordShiftLeft, km.WordShiftRight,
		km.Home, km.End,
		km.Backspace, km.Delete, km.DeleteWordBackward, km.KillLineRight, km.Enter,
		km.Undo, km.Redo,
	}
}

// isZero returns true when no bindings have been configured.
func (km KeyMap) isZero() bool {
	return allBindingsZero(km.bindings())
}

func allBindingsZero(bindings []key.Binding) bool {
	for _, b := range bindings {
		if len(b.Keys()) > 0 {
			return false
		}
	}
	return true
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left:  key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
		Right: key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
		Up:    key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
		Down:  key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		ParagraphUp: key.NewBinding(
			key.WithKeys("alt+up", "ctrl+up"),
			key.WithHelp("alt/ctrl+↑", "prev empty row"),
		),
		ParagraphDown: key.NewBinding(
			key.WithKeys("alt+down", "ctrl+down"),
			key.WithHelp("alt/ctrl+↓", "next empty row"),
		),
		PageUp:   key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "page down")),

		ShiftLeft:  key.NewBinding(key.WithKeys("shift+left"), key.WithHelp("shift+←", "select left")),
		ShiftRight: key.NewBinding(key.WithKeys("shift+right"), key.WithHelp("shift+→", "select right")),
		ShiftUp:    key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+↑", "select up")),
		ShiftDown:  key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "select down")),
		ParagraphShiftUp: key.NewBinding(
			key.WithKeys("alt+shift+up"),
			key.WithHelp("alt+shift+↑", "select prev empty row"),
		),
		ParagraphShiftDown: key.NewBinding(
			key.WithKeys("alt+shift+down"),
			key.WithHelp("alt+shift+↓", "select next empty row"),
		),

		// Portable word movement: terminals vary between alt+arrows and ctrl+arrows.
		WordLeft:       key.NewBinding(key.WithKeys("alt+left", "ctrl+left"), key.WithHelp("alt/ctrl+←", "word left")),
		WordRight:      key.NewBinding(key.WithKeys("alt+right", "ctrl+right"), key.WithHelp("alt/ctrl+→", "word right")),
		WordShiftLeft:  key.NewBinding(key.WithKeys("alt+shift+left"), key.WithHelp("alt+shift+←", "select word left")),
		WordShiftRight: key.NewBinding(key.WithKeys("alt+shift+right"), key.WithHelp("alt+shift+→", "select word right")),

		Home: key.NewBinding(key.WithKeys("home", "ctrl+a"), key.WithHelp("home", "line start")),
		End:  key.NewBinding(key.WithKeys("end", "ctrl+e"), key.WithHelp("end", "line end")),

		Backspace:          key.NewBinding(key.WithKeys("backspace", "ctrl+h"), key.WithHelp("backspace", "delete left")),
		Delete:             key.NewBinding(key.WithKeys("delete"), key.WithHelp("del", "delete right")),
		DeleteWordBackward: key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w"), key.WithHelp("opt+⌫/ctrl+w", "delete word left")),
		KillLineRight:      key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "delete line right")),
		Enter:              key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "newline")),

		Undo: key.NewBinding(key.WithKeys("ctrl+z"), key.WithHelp("ctrl+z", "undo")),
		Redo: key.NewBinding(key.WithKeys("ctrl+y", "ctrl+shift+z"), key.WithHelp("ctrl+y", "redo")),
	}
}
