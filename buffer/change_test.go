package buffer

import "testing"

func TestBuffer_LastChange_InitialAndNoOp(t *testing.T) {
	b := New("a", Options{})

	if _, ok := b.LastChange(); ok {
		t.Fatalf("expected no initial change")
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirLeft}) // no-op at BOF
	if _, ok := b.LastChange(); ok {
		t.Fatalf("expected no change after no-op mutation")
	}
}

func TestBuffer_Change_InsertTextShape(t *testing.T) {
	b := New("ab", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})
	v := b.Version()

	b.InsertText("X")

	ch, ok := b.LastChange()
	if !ok {
		t.Fatalf("expected last change")
	}
	if got, want := ch.Source, ChangeSourceLocal; got != want {
		t.Fatalf("source=%v, want %v", got, want)
	}
	if got, want := ch.VersionBefore, v; got != want {
		t.Fatalf("version before=%d, want %d", got, want)
	}
	if got, want := ch.VersionAfter, v+1; got != want {
		t.Fatalf("version after=%d, want %d", got, want)
	}
	if got, want := ch.CursorBefore, (Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor before=%v, want %v", got, want)
	}
	if got, want := ch.CursorAfter, (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor after=%v, want %v", got, want)
	}
	if ch.SelectionBefore.Active {
		t.Fatalf("expected inactive selection before")
	}
	if ch.SelectionAfter.Active {
		t.Fatalf("expected inactive selection after")
	}
	if got, want := len(ch.AppliedEdits), 1; got != want {
		t.Fatalf("applied edits=%d, want %d", got, want)
	}
	edit := ch.AppliedEdits[0]
	if got, want := edit.RangeBefore, (Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 1}}); got != want {
		t.Fatalf("range before=%v, want %v", got, want)
	}
	if got, want := edit.RangeAfter, (Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 2}}); got != want {
		t.Fatalf("range after=%v, want %v", got, want)
	}
	if got, want := edit.InsertText, "X"; got != want {
		t.Fatalf("insert text=%q, want %q", got, want)
	}
	if got, want := edit.DeletedText, ""; got != want {
		t.Fatalf("deleted text=%q, want %q", got, want)
	}
}

func TestBuffer_Change_ApplyKeepsEditOrder(t *testing.T) {
	b := New("hello", Options{})
	v := b.Version()

	b.Apply(
		TextEdit{Range: Range{Start: Pos{Row: 0, GraphemeCol: 0}, End: Pos{Row: 0, GraphemeCol: 0}}, Text: "X"},
		TextEdit{Range: Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 2}}, Text: ""},
	)

	ch, ok := b.LastChange()
	if !ok {
		t.Fatalf("expected last change")
	}
	if got, want := ch.VersionBefore, v; got != want {
		t.Fatalf("version before=%d, want %d", got, want)
	}
	if got, want := ch.VersionAfter, v+1; got != want {
		t.Fatalf("version after=%d, want %d", got, want)
	}
	if got, want := len(ch.AppliedEdits), 2; got != want {
		t.Fatalf("applied edits=%d, want %d", got, want)
	}
	first := ch.AppliedEdits[0]
	if got, want := first.InsertText, "X"; got != want {
		t.Fatalf("first insert=%q, want %q", got, want)
	}
	if got, want := first.DeletedText, ""; got != want {
		t.Fatalf("first deleted=%q, want %q", got, want)
	}
	second := ch.AppliedEdits[1]
	if got, want := second.InsertText, ""; got != want {
		t.Fatalf("second insert=%q, want %q", got, want)
	}
	if got, want := second.DeletedText, "h"; got != want {
		t.Fatalf("second deleted=%q, want %q", got, want)
	}
}

func TestBuffer_Change_MoveHasNoAppliedEdits(t *testing.T) {
	b := New("ab", Options{})
	v := b.Version()

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight})

	ch, ok := b.LastChange()
	if !ok {
		t.Fatalf("expected last change")
	}
	if got, want := ch.VersionBefore, v; got != want {
		t.Fatalf("version before=%d, want %d", got, want)
	}
	if got, want := ch.VersionAfter, v+1; got != want {
		t.Fatalf("version after=%d, want %d", got, want)
	}
	if got := len(ch.AppliedEdits); got != 0 {
		t.Fatalf("applied edits=%d, want 0", got)
	}
}

func TestBuffer_Change_NoOpDoesNotReplaceLastChange(t *testing.T) {
	b := New("ab", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 2})

	before, ok := b.LastChange()
	if !ok {
		t.Fatalf("expected last change after SetCursor")
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight}) // no-op at EOL

	after, ok := b.LastChange()
	if !ok {
		t.Fatalf("expected last change to remain available")
	}
	if got, want := after.VersionAfter, before.VersionAfter; got != want {
		t.Fatalf("version after=%d, want %d", got, want)
	}
	if got, want := after.CursorAfter, before.CursorAfter; got != want {
		t.Fatalf("cursor after=%v, want %v", got, want)
	}
}

func TestBuffer_Change_UndoRedoEmitReplacementEdit(t *testing.T) {
	b := New("", Options{})
	b.InsertText("abc")

	if ok := b.Undo(); !ok {
		t.Fatalf("expected Undo=true")
	}
	undoChange, ok := b.LastChange()
	if !ok {
		t.Fatalf("expected undo change")
	}
	if got, want := len(undoChange.AppliedEdits), 1; got != want {
		t.Fatalf("undo applied edits=%d, want %d", got, want)
	}
	undoEdit := undoChange.AppliedEdits[0]
	if got, want := undoEdit.DeletedText, "abc"; got != want {
		t.Fatalf("undo deleted=%q, want %q", got, want)
	}
	if got, want := undoEdit.InsertText, ""; got != want {
		t.Fatalf("undo insert=%q, want %q", got, want)
	}

	if ok := b.Redo(); !ok {
		t.Fatalf("expected Redo=true")
	}
	redoChange, ok := b.LastChange()
	if !ok {
		t.Fatalf("expected redo change")
	}
	if got, want := len(redoChange.AppliedEdits), 1; got != want {
		t.Fatalf("redo applied edits=%d, want %d", got, want)
	}
	redoEdit := redoChange.AppliedEdits[0]
	if got, want := redoEdit.DeletedText, ""; got != want {
		t.Fatalf("redo deleted=%q, want %q", got, want)
	}
	if got, want := redoEdit.InsertText, "abc"; got != want {
		t.Fatalf("redo insert=%q, want %q", got, want)
	}
}
