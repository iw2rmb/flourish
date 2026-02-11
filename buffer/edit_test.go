package buffer

import "testing"

func TestBuffer_InsertText_MultiLine(t *testing.T) {
	b := New("ab", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})
	v := b.Version()

	b.InsertText("X\nY")
	if got, want := b.Text(), "aX\nYb"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 1, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_InsertText_ReplacesSelection(t *testing.T) {
	b := New("hello", Options{})
	b.SetSelection(Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 4}}) // "ell"
	v := b.Version()

	b.InsertText("i")
	if got, want := b.Text(), "hio"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_InsertGrapheme_Unicode(t *testing.T) {
	b := New("", Options{})
	b.InsertGrapheme("π")
	b.InsertGrapheme("テ")

	if got, want := b.Text(), "πテ"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
}

func TestBuffer_DeleteBackward_JoinsLinesAtSOL(t *testing.T) {
	b := New("ab\ncd", Options{})
	b.SetCursor(Pos{Row: 1, GraphemeCol: 0})
	v := b.Version()

	b.DeleteBackward()
	if got, want := b.Text(), "abcd"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_DeleteForward_JoinsLinesAtEOL(t *testing.T) {
	b := New("ab\ncd", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 2})
	v := b.Version()

	b.DeleteForward()
	if got, want := b.Text(), "abcd"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_DeleteSelection_SpanningMultipleLines(t *testing.T) {
	b := New("ab\ncd\nef", Options{})
	b.SetSelection(Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 2, GraphemeCol: 1}})
	v := b.Version()

	b.DeleteSelection()
	if got, want := b.Text(), "af"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_Delete_SelectionFirstSemantics(t *testing.T) {
	b := New("abcd", Options{})
	b.SetSelection(Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 3}}) // "bc"
	v := b.Version()

	b.DeleteBackward()
	if got, want := b.Text(), "ad"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_Delete_NoOpsDoNotBumpVersion(t *testing.T) {
	b := New("a", Options{})
	v := b.Version()

	b.DeleteBackward()
	if got := b.Version(); got != v {
		t.Fatalf("version=%d, want %d", got, v)
	}

	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})
	v2 := b.Version()
	b.DeleteForward()
	if got := b.Version(); got != v2 {
		t.Fatalf("version=%d, want %d", got, v2)
	}
}

func TestBuffer_DeleteBackward_RemovesWholeGraphemeCluster(t *testing.T) {
	b := New("e\u0301x", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})

	b.DeleteBackward()
	if got, want := b.Text(), "x"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
}
