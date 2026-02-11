package buffer

import "testing"

func TestBuffer_UndoRedo_BasicTyping(t *testing.T) {
	b := New("", Options{})
	if b.CanUndo() {
		t.Fatalf("expected CanUndo=false")
	}
	if b.CanRedo() {
		t.Fatalf("expected CanRedo=false")
	}

	b.InsertGrapheme("a")
	if !b.CanUndo() {
		t.Fatalf("expected CanUndo=true")
	}
	if b.CanRedo() {
		t.Fatalf("expected CanRedo=false")
	}

	v := b.Version()
	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if got, want := b.Text(), ""; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
	if !b.CanRedo() {
		t.Fatalf("expected CanRedo=true")
	}

	v2 := b.Version()
	if ok := b.Redo(); !ok {
		t.Fatalf("expected Redo=true")
	}
	if got, want := b.Text(), "a"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if got := b.Version(); got != v2+1 {
		t.Fatalf("version=%d, want %d", got, v2+1)
	}
}

func TestBuffer_UndoRedo_EmptyStacks_NoMutation(t *testing.T) {
	b := New("hi", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})

	text := b.Text()
	cursor := b.Cursor()
	v := b.Version()

	if ok := b.Undo(); ok {
		t.Fatalf("expected Undo=false")
	}
	if ok := b.Redo(); ok {
		t.Fatalf("expected Redo=false")
	}

	if got := b.Text(); got != text {
		t.Fatalf("text=%q, want %q", got, text)
	}
	if got := b.Cursor(); got != cursor {
		t.Fatalf("cursor=%v, want %v", got, cursor)
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected no selection")
	}
	if got := b.Version(); got != v {
		t.Fatalf("version=%d, want %d", got, v)
	}
}

func TestBuffer_Undo_RestoresCursorAndSelection(t *testing.T) {
	b := New("hello", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 4})
	b.SetSelection(Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 4}}) // "ell"

	b.InsertText("i")
	if got, want := b.Text(), "hio"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared after insert")
	}

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if got, want := b.Text(), "hello"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 4}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	r, ok := b.Selection()
	if !ok {
		t.Fatalf("expected selection restored")
	}
	if got, want := r, (Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 4}}); got != want {
		t.Fatalf("selection=%v, want %v", got, want)
	}
}

func TestBuffer_Undo_RestoresSelectionAnchorDirection(t *testing.T) {
	b := New("abcd", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})
	b.SetSelection(Range{Start: Pos{Row: 0, GraphemeCol: 3}, End: Pos{Row: 0, GraphemeCol: 1}}) // anchor right-to-left

	b.InsertText("X")
	if got, want := b.Text(), "aXd"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if got, want := b.Text(), "abcd"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight, Extend: true})
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	r, ok := b.Selection()
	if !ok {
		t.Fatalf("expected selection")
	}
	if got, want := r, (Range{Start: Pos{Row: 0, GraphemeCol: 2}, End: Pos{Row: 0, GraphemeCol: 3}}); got != want {
		t.Fatalf("selection=%v, want %v", got, want)
	}
}

func TestBuffer_UndoRedo_ApplyIsSingleHistoryStep(t *testing.T) {
	b := New("ab", Options{})
	b.Apply(
		TextEdit{Range: Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 1}}, Text: "X\nY"},
		TextEdit{Range: Range{Start: Pos{Row: 1, GraphemeCol: 1}, End: Pos{Row: 1, GraphemeCol: 1}}, Text: "Z"},
	)

	if got, want := b.Text(), "aX\nYZb"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if !b.CanUndo() {
		t.Fatalf("expected CanUndo=true")
	}

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if got, want := b.Text(), "ab"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if b.CanUndo() {
		t.Fatalf("expected CanUndo=false (Apply is one history step)")
	}
	if !b.CanRedo() {
		t.Fatalf("expected CanRedo=true")
	}

	if ok := b.Redo(); !ok {
		t.Fatalf("expected Redo=true")
	}
	if got, want := b.Text(), "aX\nYZb"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
}

func TestBuffer_HistoryLimit_BoundsUndoDepth(t *testing.T) {
	b := New("", Options{HistoryLimit: 2})
	b.InsertText("a")
	b.InsertText("b")
	b.InsertText("c")

	if got, want := b.Text(), "abc"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if got, want := b.Text(), "ab"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if got, want := b.Text(), "a"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}

	if ok := b.Undo(); ok {
		t.Fatalf("expected Undo=false (history limit reached)")
	}
}

func TestBuffer_UndoThenNewEdit_ClearsRedo(t *testing.T) {
	b := New("", Options{})
	b.InsertText("a")
	b.InsertText("b")

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if !b.CanRedo() {
		t.Fatalf("expected CanRedo=true")
	}

	b.InsertText("X")
	if b.CanRedo() {
		t.Fatalf("expected CanRedo=false after new edit")
	}
}

func TestBuffer_UndoRedo_DeleteBackward_JoinsLines_Unicode(t *testing.T) {
	b := New("π\nテ", Options{})
	b.SetCursor(Pos{Row: 1, GraphemeCol: 0})

	b.DeleteBackward()
	if got, want := b.Text(), "πテ"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	if got, want := b.Text(), "π\nテ"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 1, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}

	if ok := b.Redo(); !ok {
		t.Fatalf("expected Redo=true")
	}
	if got, want := b.Text(), "πテ"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
}
