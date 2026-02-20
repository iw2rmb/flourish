package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
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

func TestCompletionPopupAccept_ReplacesProWithProject(t *testing.T) {
	m := newModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlAt})
	m = updated.(model)

	for i := 0; i < 3; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	lines := strings.Split(m.editor.Buffer().Text(), "\n")
	if got, want := lines[10], "\tproject"; got != want {
		t.Fatalf("accept project should replace pro: got %q, want %q", got, want)
	}
}

func TestCompletionPopupAccept_ReplacesProWithProfileLiteral(t *testing.T) {
	m := newModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlAt})
	m = updated.(model)

	for i := 0; i < 2; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	lines := strings.Split(m.editor.Buffer().Text(), "\n")
	if got, want := lines[10], "\tprofile{}"; got != want {
		t.Fatalf("accept profile should replace pro: got %q, want %q", got, want)
	}
}

func TestCompletionPopupAccept_FromMiddleOfPro_ReplacesWholeIdentifier(t *testing.T) {
	m := newModel()
	m.editor.Buffer().SetCursor(buffer.Pos{Row: 10, GraphemeCol: 2})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlAt})
	m = updated.(model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	lines := strings.Split(m.editor.Buffer().Text(), "\n")
	if got, want := lines[10], "\tproperty"; got != want {
		t.Fatalf("accept property from middle should replace pro: got %q, want %q", got, want)
	}
}
