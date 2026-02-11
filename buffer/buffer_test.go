package buffer

import "testing"

func TestBuffer_SetCursor_ClampsAndVersions(t *testing.T) {
	b := New("a\nbc", Options{})
	if b.Version() != 0 {
		t.Fatalf("expected version 0, got %d", b.Version())
	}

	b.SetCursor(Pos{Row: 999, GraphemeCol: 999})
	if got := b.Cursor(); got != (Pos{Row: 1, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want (1,2)", got)
	}
	if b.Version() != 1 {
		t.Fatalf("expected version 1, got %d", b.Version())
	}

	b.SetCursor(Pos{Row: 1, GraphemeCol: 2})
	if b.Version() != 1 {
		t.Fatalf("expected version unchanged, got %d", b.Version())
	}
}

func TestBuffer_SetSelection_NormalizesClampsAndVersions(t *testing.T) {
	b := New("a\nbc", Options{})

	b.SetSelection(Range{
		Start: Pos{Row: 1, GraphemeCol: 99},
		End:   Pos{Row: 0, GraphemeCol: -1},
	})

	r, ok := b.Selection()
	if !ok {
		t.Fatalf("expected selection active")
	}
	want := Range{Start: Pos{Row: 0, GraphemeCol: 0}, End: Pos{Row: 1, GraphemeCol: 2}}
	if r != want {
		t.Fatalf("selection=%v, want %v", r, want)
	}
	if b.Version() != 1 {
		t.Fatalf("expected version 1, got %d", b.Version())
	}

	// Setting the same effective selection should not bump the version.
	b.SetSelection(Range{Start: Pos{Row: 1, GraphemeCol: 2}, End: Pos{Row: 0, GraphemeCol: 0}})
	if b.Version() != 1 {
		t.Fatalf("expected version unchanged, got %d", b.Version())
	}

	b.ClearSelection()
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if b.Version() != 2 {
		t.Fatalf("expected version 2, got %d", b.Version())
	}

	// Clearing again should be a no-op.
	b.ClearSelection()
	if b.Version() != 2 {
		t.Fatalf("expected version unchanged, got %d", b.Version())
	}
}

func TestBuffer_SelectionRaw_PreservesDirection(t *testing.T) {
	b := New("abcd", Options{})

	b.SetSelection(Range{Start: Pos{Row: 0, GraphemeCol: 3}, End: Pos{Row: 0, GraphemeCol: 1}})

	raw, ok := b.SelectionRaw()
	if !ok {
		t.Fatalf("expected raw selection active")
	}
	wantRaw := Range{Start: Pos{Row: 0, GraphemeCol: 3}, End: Pos{Row: 0, GraphemeCol: 1}}
	if raw != wantRaw {
		t.Fatalf("raw selection=%v, want %v", raw, wantRaw)
	}

	norm, ok := b.Selection()
	if !ok {
		t.Fatalf("expected normalized selection active")
	}
	wantNorm := Range{Start: Pos{Row: 0, GraphemeCol: 1}, End: Pos{Row: 0, GraphemeCol: 3}}
	if norm != wantNorm {
		t.Fatalf("normalized selection=%v, want %v", norm, wantNorm)
	}
}
