package editor

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the editor key bindings.
//
// Bindings must be portable across terminals (ctrl/alt fallbacks).
type KeyMap struct {
	Left, Right, Up, Down                     key.Binding
	PageUp, PageDown                          key.Binding
	ShiftLeft, ShiftRight, ShiftUp, ShiftDown key.Binding
	WordLeft, WordRight                       key.Binding
	WordShiftLeft, WordShiftRight             key.Binding
	Home, End                                 key.Binding

	Backspace, Delete key.Binding
	Enter             key.Binding

	Undo, Redo       key.Binding
	Copy, Cut, Paste key.Binding
}

// isZero returns true when no bindings have been configured.
func (km KeyMap) isZero() bool {
	return bindingIsZero(km.Left) &&
		bindingIsZero(km.Right) &&
		bindingIsZero(km.Up) &&
		bindingIsZero(km.Down) &&
		bindingIsZero(km.PageUp) &&
		bindingIsZero(km.PageDown) &&
		bindingIsZero(km.ShiftLeft) &&
		bindingIsZero(km.ShiftRight) &&
		bindingIsZero(km.ShiftUp) &&
		bindingIsZero(km.ShiftDown) &&
		bindingIsZero(km.WordLeft) &&
		bindingIsZero(km.WordRight) &&
		bindingIsZero(km.WordShiftLeft) &&
		bindingIsZero(km.WordShiftRight) &&
		bindingIsZero(km.Home) &&
		bindingIsZero(km.End) &&
		bindingIsZero(km.Backspace) &&
		bindingIsZero(km.Delete) &&
		bindingIsZero(km.Enter) &&
		bindingIsZero(km.Undo) &&
		bindingIsZero(km.Redo) &&
		bindingIsZero(km.Copy) &&
		bindingIsZero(km.Cut) &&
		bindingIsZero(km.Paste)
}

func bindingIsZero(b key.Binding) bool {
	return len(b.Keys()) == 0
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left:     key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
		Right:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
		Up:       key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
		Down:     key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		PageUp:   key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "page down")),

		ShiftLeft:  key.NewBinding(key.WithKeys("shift+left"), key.WithHelp("shift+←", "select left")),
		ShiftRight: key.NewBinding(key.WithKeys("shift+right"), key.WithHelp("shift+→", "select right")),
		ShiftUp:    key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+↑", "select up")),
		ShiftDown:  key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "select down")),

		// Portable word movement: terminals vary between alt+arrows and ctrl+arrows.
		WordLeft:       key.NewBinding(key.WithKeys("alt+left", "ctrl+left"), key.WithHelp("alt/ctrl+←", "word left")),
		WordRight:      key.NewBinding(key.WithKeys("alt+right", "ctrl+right"), key.WithHelp("alt/ctrl+→", "word right")),
		WordShiftLeft:  key.NewBinding(key.WithKeys("alt+shift+left", "ctrl+shift+left"), key.WithHelp("alt/ctrl+shift+←", "select word left")),
		WordShiftRight: key.NewBinding(key.WithKeys("alt+shift+right", "ctrl+shift+right"), key.WithHelp("alt/ctrl+shift+→", "select word right")),

		Home: key.NewBinding(key.WithKeys("home", "ctrl+a"), key.WithHelp("home", "line start")),
		End:  key.NewBinding(key.WithKeys("end", "ctrl+e"), key.WithHelp("end", "line end")),

		Backspace: key.NewBinding(key.WithKeys("backspace", "ctrl+h"), key.WithHelp("backspace", "delete left")),
		Delete:    key.NewBinding(key.WithKeys("delete"), key.WithHelp("del", "delete right")),
		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "newline")),

		Undo: key.NewBinding(key.WithKeys("ctrl+z"), key.WithHelp("ctrl+z", "undo")),
		Redo: key.NewBinding(key.WithKeys("ctrl+y", "ctrl+shift+z"), key.WithHelp("ctrl+y", "redo")),

		Copy:  key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "copy")),
		Cut:   key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "cut")),
		Paste: key.NewBinding(key.WithKeys("ctrl+v"), key.WithHelp("ctrl+v", "paste")),
	}
}
