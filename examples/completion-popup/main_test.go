package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCompletionPopupArrowDownNavigatesSelection(t *testing.T) {
	m := newModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlAt})
	m = updated.(model)

	state := m.editor.CompletionState()
	if !state.Visible {
		t.Fatalf("completion should be visible after trigger")
	}
	if got, want := state.Selected, 0; got != want {
		t.Fatalf("initial selected index: got %d, want %d", got, want)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)

	state = m.editor.CompletionState()
	if got, want := state.Selected, 1; got != want {
		t.Fatalf("selected index after down: got %d, want %d", got, want)
	}
}
