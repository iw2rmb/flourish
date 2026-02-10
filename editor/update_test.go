package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flouris/buffer"
)

type memClipboard struct {
	t *testing.T
	s string
}

func (c *memClipboard) ReadText() (string, error) { return c.s, nil }
func (c *memClipboard) WriteText(s string) error  { c.s = s; return nil }

func TestUpdate_TypingMovementAndDelete(t *testing.T) {
	m := New(Config{
		Text:  "ab",
		Style: Style{}, // keep styles minimal for this test
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if got := m.buf.Text(); got != "aXb" {
		t.Fatalf("text after insert: got %q, want %q", got, "aXb")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, Col: 2}) {
		t.Fatalf("cursor after insert: got %v, want %v", got, buffer.Pos{Row: 0, Col: 2})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after backspace: got %q, want %q", got, "ab")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, Col: 1}) {
		t.Fatalf("cursor after backspace: got %v, want %v", got, buffer.Pos{Row: 0, Col: 1})
	}
}

func TestUpdate_ReadOnly_IgnoresMutations(t *testing.T) {
	m := New(Config{
		Text:     "ab",
		ReadOnly: true,
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, Col: 1}) {
		t.Fatalf("cursor after move: got %v, want %v", got, buffer.Pos{Row: 0, Col: 1})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after insert in read-only: got %q, want %q", got, "ab")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, Col: 1}) {
		t.Fatalf("cursor after insert in read-only: got %v, want %v", got, buffer.Pos{Row: 0, Col: 1})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after backspace in read-only: got %q, want %q", got, "ab")
	}
}

func TestUpdate_UndoRedo(t *testing.T) {
	m := New(Config{Text: ""})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after typing: got %q, want %q", got, "ab")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if got := m.buf.Text(); got != "a" {
		t.Fatalf("text after undo: got %q, want %q", got, "a")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after redo: got %q, want %q", got, "ab")
	}
}

func TestUpdate_CopyCutPaste(t *testing.T) {
	cb := &memClipboard{t: t}
	m := New(Config{
		Text:      "hello",
		Clipboard: cb,
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftRight})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if got := cb.s; got != "he" {
		t.Fatalf("clipboard after copy: got %q, want %q", got, "he")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	if got := m.buf.Text(); got != "llo" {
		t.Fatalf("text after cut: got %q, want %q", got, "llo")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, Col: 0}) {
		t.Fatalf("cursor after cut: got %v, want %v", got, buffer.Pos{Row: 0, Col: 0})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	if got := m.buf.Text(); got != "hello" {
		t.Fatalf("text after paste: got %q, want %q", got, "hello")
	}
}

func TestUpdate_ViewportFollowsCursor_Minimal(t *testing.T) {
	m := New(Config{Text: "0\n1\n2\n3\n4\n5\n6\n7\n8\n9"})
	m = m.SetSize(10, 3)

	if got := m.viewport.YOffset; got != 0 {
		t.Fatalf("initial yoffset: got %d, want %d", got, 0)
	}

	// Move to row 2: still visible, no scroll.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := m.viewport.YOffset; got != 0 {
		t.Fatalf("yoffset at row 2: got %d, want %d", got, 0)
	}

	// Move to row 3: scroll down by one line.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := m.viewport.YOffset; got != 1 {
		t.Fatalf("yoffset at row 3: got %d, want %d", got, 1)
	}

	// Move to row 4: scroll down by one more line.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := m.viewport.YOffset; got != 2 {
		t.Fatalf("yoffset at row 4: got %d, want %d", got, 2)
	}

	// Move up within view: no scroll.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := m.viewport.YOffset; got != 2 {
		t.Fatalf("yoffset after up within view: got %d, want %d", got, 2)
	}

	// Move up above the viewport: yoffset follows cursor row.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}) // row 2
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}) // row 1
	if got := m.viewport.YOffset; got != 1 {
		t.Fatalf("yoffset after moving above view: got %d, want %d", got, 1)
	}
}
