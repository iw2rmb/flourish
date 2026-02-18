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
	if got := events[0].Change.Source; got != buffer.ChangeSourceLocal {
		t.Fatalf("event source after move: got %v, want %v", got, buffer.ChangeSourceLocal)
	}
	if got := events[0].Change.VersionBefore; got != 0 {
		t.Fatalf("event version before after move: got %d, want %d", got, 0)
	}
	if got := events[0].Change.VersionAfter; got != 1 {
		t.Fatalf("event version after move: got %d, want %d", got, 1)
	}
	if got := events[0].Change.CursorBefore; got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("event cursor before after move: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}
	if got := events[0].Change.CursorAfter; got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("event cursor after move: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}
	if got := len(events[0].Change.AppliedEdits); got != 0 {
		t.Fatalf("event applied edits after move: got %d, want %d", got, 0)
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
	if got := events[2].Change.VersionBefore; got != 2 {
		t.Fatalf("event version before after insert: got %d, want %d", got, 2)
	}
	if got := events[2].Change.VersionAfter; got != 3 {
		t.Fatalf("event version after insert: got %d, want %d", got, 3)
	}
	if got := events[2].Change.CursorBefore; got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("event cursor before after insert: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
	if got := events[2].Change.CursorAfter; got != (buffer.Pos{Row: 0, GraphemeCol: 3}) {
		t.Fatalf("event cursor after insert: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 3})
	}
	if got := len(events[2].Change.AppliedEdits); got != 1 {
		t.Fatalf("event applied edits after insert: got %d, want %d", got, 1)
	}
	edit := events[2].Change.AppliedEdits[0]
	if got := edit.RangeBefore; got != (buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 2},
		End:   buffer.Pos{Row: 0, GraphemeCol: 2},
	}) {
		t.Fatalf("event edit range before after insert: got %v", got)
	}
	if got := edit.RangeAfter; got != (buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 2},
		End:   buffer.Pos{Row: 0, GraphemeCol: 3},
	}) {
		t.Fatalf("event edit range after after insert: got %v", got)
	}
	if got := edit.InsertText; got != "X" {
		t.Fatalf("event edit insert text after insert: got %q, want %q", got, "X")
	}
	if got := edit.DeletedText; got != "" {
		t.Fatalf("event edit deleted text after insert: got %q, want empty", got)
	}
}
