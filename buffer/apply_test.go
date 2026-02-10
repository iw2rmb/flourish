package buffer

import "testing"

func TestBuffer_Apply_AppliesSequentiallyAgainstEvolvingState(t *testing.T) {
	b := New("hello", Options{})
	v := b.Version()

	b.Apply(
		TextEdit{Range: Range{Start: Pos{Row: 0, Col: 0}, End: Pos{Row: 0, Col: 0}}, Text: "X"},
		TextEdit{Range: Range{Start: Pos{Row: 0, Col: 1}, End: Pos{Row: 0, Col: 2}}, Text: ""},
	)

	if got, want := b.Text(), "Xello"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, Col: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_Apply_ClampsOutOfBoundsRanges(t *testing.T) {
	b := New("ab\ncd", Options{})
	v := b.Version()

	b.Apply(
		TextEdit{Range: Range{Start: Pos{Row: 999, Col: 999}, End: Pos{Row: 999, Col: 999}}, Text: "X"},
		TextEdit{Range: Range{Start: Pos{Row: -5, Col: -9}, End: Pos{Row: -5, Col: -9}}, Text: "Y"},
	)

	if got, want := b.Text(), "Yab\ncdX"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, Col: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if got := b.Version(); got != v+1 {
		t.Fatalf("version=%d, want %d", got, v+1)
	}
}

func TestBuffer_Apply_MultiLineInsertAndFollowupEdit(t *testing.T) {
	b := New("a", Options{})

	b.Apply(
		TextEdit{Range: Range{Start: Pos{Row: 0, Col: 1}, End: Pos{Row: 0, Col: 1}}, Text: "\n"},
		TextEdit{Range: Range{Start: Pos{Row: 1, Col: 0}, End: Pos{Row: 1, Col: 0}}, Text: "b"},
	)

	if got, want := b.Text(), "a\nb"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 1, Col: 1}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
}

func TestBuffer_Apply_NoOpDoesNotBumpVersion(t *testing.T) {
	b := New("a", Options{})
	v := b.Version()

	b.Apply(TextEdit{Range: Range{Start: Pos{Row: 0, Col: 0}, End: Pos{Row: 0, Col: 0}}, Text: ""})
	if got := b.Version(); got != v {
		t.Fatalf("version=%d, want %d", got, v)
	}
}
