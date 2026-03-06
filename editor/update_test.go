package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/iw2rmb/flourish/buffer"
)

func TestUpdate_TypingMovementAndDelete(t *testing.T) {
	m := New(Config{
		Text:  "ab",
		Style: Style{}, // keep styles minimal for this test
	})

	m, _ = m.Update(testKeyCode(tea.KeyRight))
	m, _ = m.Update(testKeyText("X"))
	if got := m.buf.Text(); got != "aXb" {
		t.Fatalf("text after insert: got %q, want %q", got, "aXb")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor after insert: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}

	m, _ = m.Update(testKeyCode(tea.KeyBackspace))
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after backspace: got %q, want %q", got, "ab")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor after backspace: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}
}

func TestUpdate_SpaceKey_InsertsSpace(t *testing.T) {
	m := New(Config{Text: "ab"})
	m, _ = m.Update(testKeyCode(tea.KeyRight))
	m, _ = m.Update(testKeyCode(tea.KeySpace))

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

	m, _ = m.Update(testKeyCode(tea.KeyRight))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor after move: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}

	m, _ = m.Update(testKeyText("X"))
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after insert in read-only: got %q, want %q", got, "ab")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor after insert in read-only: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}

	m, _ = m.Update(testKeyCode(tea.KeyBackspace))
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after backspace in read-only: got %q, want %q", got, "ab")
	}
}

func TestUpdate_UndoRedo(t *testing.T) {
	m := New(Config{Text: ""})
	m, _ = m.Update(testKeyText("a"))
	m, _ = m.Update(testKeyText("b"))
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after typing: got %q, want %q", got, "ab")
	}

	m, _ = m.Update(testKeyCode('z', tea.ModCtrl))
	if got := m.buf.Text(); got != "a" {
		t.Fatalf("text after undo: got %q, want %q", got, "a")
	}

	m, _ = m.Update(testKeyCode('y', tea.ModCtrl))
	if got := m.buf.Text(); got != "ab" {
		t.Fatalf("text after redo: got %q, want %q", got, "ab")
	}
}

func TestUpdate_OptLeftRight_JumpsByWord(t *testing.T) {
	m := New(Config{Text: "alpha beta gamma"})

	m, _ = m.Update(testKeyCode(tea.KeyRight, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("cursor after opt+right from col 0: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}

	m, _ = m.Update(testKeyCode(tea.KeyRight, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 10}) {
		t.Fatalf("cursor after second opt+right: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 10})
	}

	m, _ = m.Update(testKeyCode(tea.KeyLeft, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("cursor after opt+left from col 10: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 6})
	}
}

func TestUpdate_OptShiftLeftRight_ExtendsAndDeselection(t *testing.T) {
	m := New(Config{Text: "alpha beta gamma"})

	m, _ = m.Update(testKeyCode(tea.KeyRight, tea.ModShift, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("cursor after opt+shift+right: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 5}}) {
		t.Fatalf("selection after opt+shift+right: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 5}}, true)
	}

	m, _ = m.Update(testKeyCode(tea.KeyRight, tea.ModShift, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 10}) {
		t.Fatalf("cursor after second opt+shift+right: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 10})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 10}}) {
		t.Fatalf("selection after second opt+shift+right: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 10}}, true)
	}

	m, _ = m.Update(testKeyCode(tea.KeyLeft, tea.ModShift, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("cursor after opt+shift+left: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 6})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 6}}) {
		t.Fatalf("selection after opt+shift+left: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 0}, End: buffer.Pos{Row: 0, GraphemeCol: 6}}, true)
	}

	m, _ = m.Update(testKeyCode(tea.KeyLeft, tea.ModShift, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor after second opt+shift+left: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("expected selection cleared after returning to anchor with opt+shift+left")
	}
}

func TestUpdate_CtrlUpDown_MovesToEmptyRows(t *testing.T) {
	m := New(Config{Text: "a\nb\n\nc\n\nend"})
	m.buf.SetCursor(buffer.Pos{Row: 1, GraphemeCol: 1})

	m, _ = m.Update(testKeyCode(tea.KeyDown, tea.ModCtrl))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 2, GraphemeCol: 0}) {
		t.Fatalf("cursor after ctrl+down: got %v, want %v", got, buffer.Pos{Row: 2, GraphemeCol: 0})
	}

	m, _ = m.Update(testKeyCode(tea.KeyDown, tea.ModCtrl))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 4, GraphemeCol: 0}) {
		t.Fatalf("cursor after second ctrl+down: got %v, want %v", got, buffer.Pos{Row: 4, GraphemeCol: 0})
	}

	m, _ = m.Update(testKeyCode(tea.KeyUp, tea.ModCtrl))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 2, GraphemeCol: 0}) {
		t.Fatalf("cursor after ctrl+up: got %v, want %v", got, buffer.Pos{Row: 2, GraphemeCol: 0})
	}

	m, _ = m.Update(testKeyCode(tea.KeyUp, tea.ModCtrl))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor after second ctrl+up: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}
}

func TestUpdate_AltUpDown_MovesToEmptyRows(t *testing.T) {
	m := New(Config{Text: "a\nb\n\nc\n\nend"})
	m.buf.SetCursor(buffer.Pos{Row: 1, GraphemeCol: 1})

	m, _ = m.Update(testKeyCode(tea.KeyDown, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 2, GraphemeCol: 0}) {
		t.Fatalf("cursor after alt+down: got %v, want %v", got, buffer.Pos{Row: 2, GraphemeCol: 0})
	}

	m, _ = m.Update(testKeyCode(tea.KeyDown, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 4, GraphemeCol: 0}) {
		t.Fatalf("cursor after second alt+down: got %v, want %v", got, buffer.Pos{Row: 4, GraphemeCol: 0})
	}

	m, _ = m.Update(testKeyCode(tea.KeyUp, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 2, GraphemeCol: 0}) {
		t.Fatalf("cursor after alt+up: got %v, want %v", got, buffer.Pos{Row: 2, GraphemeCol: 0})
	}
}

func TestUpdate_ShiftUpDown_UsesSameCursorMovementAsUpDown(t *testing.T) {
	a := New(Config{Text: "012345\nx\n012345"})
	b := New(Config{Text: "012345\nx\n012345"})
	a.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 5})
	b.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 5})

	a, _ = a.Update(testKeyCode(tea.KeyDown))
	a, _ = a.Update(testKeyCode(tea.KeyDown))

	b, _ = b.Update(testKeyCode(tea.KeyDown, tea.ModShift))
	b, _ = b.Update(testKeyCode(tea.KeyDown, tea.ModShift))

	if got, want := b.buf.Cursor(), a.buf.Cursor(); got != want {
		t.Fatalf("shift movement cursor parity: got %v, want %v", got, want)
	}
	if got, ok := b.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 5}, End: buffer.Pos{Row: 2, GraphemeCol: 5}}) {
		t.Fatalf("selection after shift-down chain: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 5}, End: buffer.Pos{Row: 2, GraphemeCol: 5}}, true)
	}
}

func TestUpdate_AltShiftUpDown_ExtendsToEmptyRows(t *testing.T) {
	m := New(Config{Text: "012345\nab\n\nxyz\n\nqwerty"})
	m.buf.SetCursor(buffer.Pos{Row: 1, GraphemeCol: 1})

	m, _ = m.Update(testKeyCode(tea.KeyDown, tea.ModShift, tea.ModAlt))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 2, GraphemeCol: 0}) {
		t.Fatalf("cursor after alt+shift+down: got %v, want %v", got, buffer.Pos{Row: 2, GraphemeCol: 0})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 1}, End: buffer.Pos{Row: 2, GraphemeCol: 0}}) {
		t.Fatalf("selection after alt+shift+down: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 1}, End: buffer.Pos{Row: 2, GraphemeCol: 0}}, true)
	}
}

func TestUpdate_CtrlShiftUpDown_NoDefaultSelectionBinding(t *testing.T) {
	m := New(Config{Text: "012345\nab\n\nxyz\n\nqwerty"})
	m.buf.SetCursor(buffer.Pos{Row: 1, GraphemeCol: 1})

	m, _ = m.Update(testKeyCode(tea.KeyDown, tea.ModShift, tea.ModCtrl))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 1, GraphemeCol: 1}) {
		t.Fatalf("cursor after ctrl+shift+down: got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 1})
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("selection after ctrl+shift+down: got active, want inactive")
	}
}

func TestUpdate_PasteMsgIgnoredByEditor(t *testing.T) {
	m := New(Config{Text: "hello"})
	m, _ = m.Update(tea.PasteMsg{Content: "X"})
	if got, want := m.buf.Text(), "hello"; got != want {
		t.Fatalf("text after paste msg: got %q, want %q", got, want)
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
	m, _ = m.Update(testMouseClick(3, 1, tea.MouseLeft))
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 1, GraphemeCol: 1}) {
		t.Fatalf("cursor after click: got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 1})
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("expected selection cleared after click")
	}

	// Shift+click extends from current cursor (anchor).
	m, _ = m.Update(testMouseClick(5, 1, tea.MouseLeft, tea.ModShift)) // col 3
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 1, GraphemeCol: 3}) {
		t.Fatalf("cursor after shift+click: got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 3})
	}
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 1}, End: buffer.Pos{Row: 1, GraphemeCol: 3}}) {
		t.Fatalf("selection after shift+click: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 1}, End: buffer.Pos{Row: 1, GraphemeCol: 3}}, true)
	}

	// Another shift+click keeps the original anchor (col 1) and updates end (col 0).
	m, _ = m.Update(testMouseClick(2, 1, tea.MouseLeft, tea.ModShift)) // col 0
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 0}, End: buffer.Pos{Row: 1, GraphemeCol: 1}}) {
		t.Fatalf("selection after second shift+click: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 1, GraphemeCol: 0}, End: buffer.Pos{Row: 1, GraphemeCol: 1}}, true)
	}

	// Drag selection: press at row 0, col 1 then motion to row 0, col 3.
	m, _ = m.Update(testMouseClick(3, 0, tea.MouseLeft))
	m, _ = m.Update(testMouseMotion(5, 0, tea.MouseNone))
	if got, ok := m.buf.Selection(); !ok || got != (buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 1}, End: buffer.Pos{Row: 0, GraphemeCol: 3}}) {
		t.Fatalf("selection after drag: got (%v,%v), want (%v,%v)", got, ok, buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 1}, End: buffer.Pos{Row: 0, GraphemeCol: 3}}, true)
	}
	m, _ = m.Update(testMouseRelease(5, 0, tea.MouseLeft))
}

func TestUpdate_ViewportFollowsCursor_Minimal(t *testing.T) {
	m := New(Config{Text: "0\n1\n2\n3\n4\n5\n6\n7\n8\n9"})
	m = m.SetSize(10, 3)

	if got := m.viewport.YOffset(); got != 0 {
		t.Fatalf("initial yoffset: got %d, want %d", got, 0)
	}

	// Move to row 2: still visible, no scroll.
	m, _ = m.Update(testKeyCode(tea.KeyDown))
	m, _ = m.Update(testKeyCode(tea.KeyDown))
	if got := m.viewport.YOffset(); got != 0 {
		t.Fatalf("yoffset at row 2: got %d, want %d", got, 0)
	}

	// Move to row 3: scroll down by one line.
	m, _ = m.Update(testKeyCode(tea.KeyDown))
	if got := m.viewport.YOffset(); got != 1 {
		t.Fatalf("yoffset at row 3: got %d, want %d", got, 1)
	}

	// Move to row 4: scroll down by one more line.
	m, _ = m.Update(testKeyCode(tea.KeyDown))
	if got := m.viewport.YOffset(); got != 2 {
		t.Fatalf("yoffset at row 4: got %d, want %d", got, 2)
	}

	// Move up within view: no scroll.
	m, _ = m.Update(testKeyCode(tea.KeyUp))
	if got := m.viewport.YOffset(); got != 2 {
		t.Fatalf("yoffset after up within view: got %d, want %d", got, 2)
	}

	// Move up above the viewport: yoffset follows cursor row.
	m, _ = m.Update(testKeyCode(tea.KeyUp)) // row 2
	m, _ = m.Update(testKeyCode(tea.KeyUp)) // row 1
	if got := m.viewport.YOffset(); got != 1 {
		t.Fatalf("yoffset after moving above view: got %d, want %d", got, 1)
	}
}

func TestUpdate_PageUpDown_MoveByVisibleRows(t *testing.T) {
	m := New(Config{Text: "0\n1\n2\n3\n4\n5\n6\n7\n8\n9"})
	m = m.SetSize(10, 3)

	m, _ = m.Update(testKeyCode(tea.KeyPgDown))
	if got, want := m.buf.Cursor(), (buffer.Pos{Row: 3, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor after pgdown: got %v, want %v", got, want)
	}

	m, _ = m.Update(testKeyCode(tea.KeyPgDown))
	if got, want := m.buf.Cursor(), (buffer.Pos{Row: 6, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor after second pgdown: got %v, want %v", got, want)
	}

	m, _ = m.Update(testKeyCode(tea.KeyPgUp))
	if got, want := m.buf.Cursor(), (buffer.Pos{Row: 3, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor after pgup: got %v, want %v", got, want)
	}
}

func TestUpdate_HorizontalScroll_FollowsCursor_LongLine(t *testing.T) {
	m := New(Config{Text: "abcdefghij"})
	m = m.SetSize(5, 1)

	for i := 0; i < 5; i++ {
		m, _ = m.Update(testKeyCode(tea.KeyRight))
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("cursor after 5 rights: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}
	if got := m.xOffset; got != 1 {
		t.Fatalf("xOffset at col 5, width 5: got %d, want %d", got, 1)
	}

	m, _ = m.Update(testKeyCode(tea.KeyRight)) // col 6
	if got := m.xOffset; got != 2 {
		t.Fatalf("xOffset at col 6, width 5: got %d, want %d", got, 2)
	}

	for i := 0; i < 6; i++ {
		m, _ = m.Update(testKeyCode(tea.KeyLeft))
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

	m, _ = m.Update(testKeyCode(tea.KeyRight)) // col 1 (tab)
	m, _ = m.Update(testKeyCode(tea.KeyRight)) // col 2 ("b")

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
		Scrollbar: ScrollbarConfig{
			Vertical:   ScrollbarNever,
			Horizontal: ScrollbarNever,
		},
	})
	m = m.SetSize(3, 2)

	for i := 0; i < 6; i++ {
		m, _ = m.Update(testKeyCode(tea.KeyRight))
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("cursor after 6 rights: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 6})
	}
	if got := m.viewport.YOffset(); got != 1 {
		t.Fatalf("yOffset after entering 3rd visual row: got %d, want %d", got, 1)
	}
	if got := m.xOffset; got != 0 {
		t.Fatalf("xOffset under soft wrap must stay 0: got %d, want %d", got, 0)
	}

	for i := 0; i < 4; i++ {
		m, _ = m.Update(testKeyCode(tea.KeyLeft))
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor after moving left: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
	if got := m.viewport.YOffset(); got != 0 {
		t.Fatalf("yOffset after returning to first visual row: got %d, want %d", got, 0)
	}
}

func TestUpdate_WrapWord_InsertAtPunctuationBoundary_AdvancesVisualCursor(t *testing.T) {
	m := New(Config{
		Text:     "abc,def,ghi",
		WrapMode: WrapWord,
	})
	m = m.SetSize(3, 6)
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 3})

	beforeX, beforeY, beforeOK := m.DocToScreen(m.buf.Cursor())
	if !beforeOK {
		t.Fatalf("before insert cursor must be visible")
	}

	m, _ = m.Update(testKeyText("x"))
	afterX, afterY, afterOK := m.DocToScreen(m.buf.Cursor())
	if !afterOK {
		t.Fatalf("after insert cursor must be visible")
	}

	if m.buf.Cursor() != (buffer.Pos{Row: 0, GraphemeCol: 4}) {
		t.Fatalf("cursor after insert: got %v, want %v", m.buf.Cursor(), buffer.Pos{Row: 0, GraphemeCol: 4})
	}
	if afterY < beforeY || (afterY == beforeY && afterX <= beforeX) {
		t.Fatalf(
			"visual cursor must advance after insert at wrap punctuation boundary: before=(%d,%d) after=(%d,%d)",
			beforeX,
			beforeY,
			afterX,
			afterY,
		)
	}
}
