package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

func TestOnChange_FiresOnMutationsAndSkipsNoOps(t *testing.T) {
	var events []ChangeEvent
	m := New(Config{
		Text: "ab",
		OnChange: func(ev ChangeEvent) {
			events = append(events, ev)
		},
	})
	events = nil

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if len(events) != 1 {
		t.Fatalf("events after move: got %d, want %d", len(events), 1)
	}
	if got := events[0].Text; got != "ab" {
		t.Fatalf("event text after move: got %q, want %q", got, "ab")
	}
	if got := events[0].Cursor; got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("event cursor after move: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // to EOL
	if len(events) != 2 {
		t.Fatalf("events after move to EOL: got %d, want %d", len(events), 2)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // no-op at EOL
	if len(events) != 2 {
		t.Fatalf("events after no-op: got %d, want %d", len(events), 2)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if len(events) != 3 {
		t.Fatalf("events after insert: got %d, want %d", len(events), 3)
	}
	if got := events[2].Text; got != "abX" {
		t.Fatalf("event text after insert: got %q, want %q", got, "abX")
	}
}
