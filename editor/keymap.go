package editor

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the editor key bindings.
//
// Bindings must be portable across terminals (ctrl/alt fallbacks).
type KeyMap struct {
	Left, Right, Up, Down                     key.Binding
	ShiftLeft, ShiftRight, ShiftUp, ShiftDown key.Binding
	WordLeft, WordRight                       key.Binding
	Home, End                                 key.Binding

	Backspace, Delete key.Binding
	Enter             key.Binding

	Undo, Redo       key.Binding
	Copy, Cut, Paste key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left:  key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
		Right: key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
		Up:    key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
		Down:  key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),

		ShiftLeft:  key.NewBinding(key.WithKeys("shift+left"), key.WithHelp("shift+←", "select left")),
		ShiftRight: key.NewBinding(key.WithKeys("shift+right"), key.WithHelp("shift+→", "select right")),
		ShiftUp:    key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+↑", "select up")),
		ShiftDown:  key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "select down")),

		// Portable word movement: terminals vary between alt+arrows and ctrl+arrows.
		WordLeft:  key.NewBinding(key.WithKeys("alt+left", "ctrl+left"), key.WithHelp("alt/ctrl+←", "word left")),
		WordRight: key.NewBinding(key.WithKeys("alt+right", "ctrl+right"), key.WithHelp("alt/ctrl+→", "word right")),

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
