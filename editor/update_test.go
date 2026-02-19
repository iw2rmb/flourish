package editor

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
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
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor after insert: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after backspace: got %q, want %q", got, "ab")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor after backspace: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}
}

func TestUpdate_SpaceKey_InsertsSpace(t *testing.T) {
	m := New(Config{Text: "ab"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	if got, want := m.buf.Text(), "a b"; got != want {
		t.Fatalf("text after space: got %q, want %q", got, want)
	}
	if got, want := m.buf.Cursor(), (buffer.Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor after space: got %v, want %v", got, want)
	}
}

func TestUpdate_ReadOnly_IgnoresMutations(t *testing.T) {
	m := New(Config{
		Text:     "ab",
		ReadOnly: true,
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor after move: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after insert in read-only: got %q, want %q", got, "ab")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor after insert in read-only: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
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

func TestUpdate_OptLeftRight_JumpsByWord(t *testing.T) {
	m := New(Config{Text: "alpha beta gamma"})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("cursor after opt+right from col 0: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 10}) {
		t.Fatalf("cursor after second opt+right: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 10})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("cursor after opt+left from col 10: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 6})
	}
}

func TestUpdate_OptShiftLeftRight_ExtendsAndDeselection(t *testing.T) {
	m := New(Config{Text: "alpha beta gamma"})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftRight, Alt: true})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("cursor after opt+shift+right: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 5}}) {
		t.Fatalf("selection after opt+shift+right: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 5}}, true)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftRight, Alt: true})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 10}) {
		t.Fatalf("cursor after second opt+shift+right: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 10})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 10}}) {
		t.Fatalf("selection after second opt+shift+right: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 10}}, true)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftLeft, Alt: true})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("cursor after opt+shift+left: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 6})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 6}}) {
		t.Fatalf("selection after opt+shift+left: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 6}}, true)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftLeft, Alt: true})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor after second opt+shift+left: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("expected selection cleared after returning to anchor with opt+shift+left")
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
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor after cut: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	if got := m.buf.Text(); got != "hello" {
		t.Fatalf("text after paste: got %q, want %q", got, "hello")
	}
}

func TestUpdate_Copy_SelectionUsesGraphemeColumns(t *testing.T) {
	cb := &memClipboard{t: t}
	m := New(Config{
		Text:      "e\u0301x",
		Clipboard: cb,
	})

	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 0},
		End:   buffer.Pos{Row: 0, GraphemeCol: 1},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if got, want := cb.s, "e\u0301"; got != want {
		t.Fatalf("clipboard after grapheme copy: got %q, want %q", got, want)
	}
}

func TestUpdate_Paste_ReplacesSelection(t *testing.T) {
	cb := &memClipboard{t: t, s: "X"}
	m := New(Config{
		Text:      "hello",
		Clipboard: cb,
	})

	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 1}, // "ell"
		End:   buffer.Pos{Row: 0, GraphemeCol: 4},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	if got := m.buf.Text(); got != "hXo" {
		t.Fatalf("text after paste replacing selection: got %q, want %q", got, "hXo")
	}
}

type errClipboard struct{}

func (c *errClipboard) ReadText() (string, error) { return "", errors.New("clipboard read error") }
func (c *errClipboard) WriteText(string) error    { return errors.New("clipboard write error") }

func TestUpdate_ClipboardErrorsIgnored(t *testing.T) {
	m := New(Config{
		Text:      "hello",
		Clipboard: &errClipboard{},
	})

	// With no selection, copy/cut are no-ops (and must not panic).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})

	// Paste read error must be ignored.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	if got := m.buf.Text(); got != "hello" {
		t.Fatalf("text after paste with clipboard error: got %q, want %q", got, "hello")
	}
}

func TestUpdate_MouseClickShiftClickAndDrag(t *testing.T) {
	m := New(Config{
		Text:   "abcd\nefgh",
		Gutter: LineNumberGutter(),
	})
	m = m.SetSize(20, 2)

	// Seed a selection; plain click should clear it.
	m.buf.SetSelection(buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 1}, End: buffer.Pos{Row: 0, GraphemeCol: 3}})

	// With line numbers: gutter width is 2 (1 digit + space). Click on row 1, col 1 => x=3, y=1.
	m, _ = m.Update(tea.MouseMsg{X: 3, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 1, GraphemeCol: 1}) {
		t.Fatalf("cursor after click: got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 1})
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("expected selection cleared after click")
	}

	// Shift+click extends from current cursor (anchor).
	m, _ = m.Update(tea.MouseMsg{X: 5, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Shift: true}) // col 3
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 1, GraphemeCol: 3}) {
		t.Fatalf("cursor after shift+click: got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 3})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 1}, End: buffer.Pos{Row: 1, GraphemeCol: 3}}) {
		t.Fatalf("selection after shift+click: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 1}, End: buffer.Pos{Row: 1, GraphemeCol: 3}}, true)
	}

	// Another shift+click keeps the original anchor (col 1) and updates end (col 0).
	m, _ = m.Update(tea.MouseMsg{X: 2, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Shift: true}) // col 0
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 0}, End: buffer.Pos{Row: 1, GraphemeCol: 1}}) {
		t.Fatalf("selection after second shift+click: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 0}, End: buffer.Pos{Row: 1, GraphemeCol: 1}}, true)
	}

	// Drag selection: press at row 0, col 1 then motion to row 0, col 3.
	m, _ = m.Update(tea.MouseMsg{X: 3, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m, _ = m.Update(tea.MouseMsg{X: 5, Y: 0, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 1}, End: buffer.Pos{Row: 0, GraphemeCol: 3}}) {
		t.Fatalf("selection after drag: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 1}, End: buffer.Pos{Row: 0, GraphemeCol: 3}}, true)
	}
	m, _ = m.Update(tea.MouseMsg{X: 5, Y: 0, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
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

func TestUpdate_HorizontalScroll_FollowsCursor_LongLine(t *testing.T) {
	m := New(Config{Text: "abcdefghij"})
	m = m.SetSize(5, 1)

	for i := 0; i < 5; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("cursor after 5 rights: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}
	if got := m.xOffset; got != 1 {
		t.Fatalf("xOffset at col 5, width 5: got %d, want %d", got, 1)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // col 6
	if got := m.xOffset; got != 2 {
		t.Fatalf("xOffset at col 6, width 5: got %d, want %d", got, 2)
	}

	for i := 0; i < 6; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor after moving back: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}
	if got := m.xOffset; got != 0 {
		t.Fatalf("xOffset after moving back into view: got %d, want %d", got, 0)
	}
}

func TestUpdate_HorizontalScroll_UsesCellCoordinates_Tab(t *testing.T) {
	m := New(Config{
		Text:     "a\tb",
		TabWidth: 4,
	})
	m = m.SetSize(3, 1)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // col 1 (tab)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // col 2 ("b")

	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor at b: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
	// Visual cells: "a" [0], tab spaces [1..3], "b" [4]. Keep cell 4 visible in width 3 => xOffset 2.
	if got := m.xOffset; got != 2 {
		t.Fatalf("xOffset at b (cell 4), width 3: got %d, want %d", got, 2)
	}
}

func TestUpdate_SoftWrap_ViewportFollowsCursorByVisualRow(t *testing.T) {
	m := New(Config{
		Text:     "0123456789",
		WrapMode: WrapGrapheme,
	})
	m = m.SetSize(3, 2)

	for i := 0; i < 6; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("cursor after 6 rights: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 6})
	}
	if got := m.viewport.YOffset; got != 1 {
		t.Fatalf("yOffset after entering 3rd visual row: got %d, want %d", got, 1)
	}
	if got := m.xOffset; got != 0 {
		t.Fatalf("xOffset under soft wrap must stay 0: got %d, want %d", got, 0)
	}

	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor after moving left: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
	if got := m.viewport.YOffset; got != 0 {
		t.Fatalf("yOffset after returning to first visual row: got %d, want %d", got, 0)
	}
}
