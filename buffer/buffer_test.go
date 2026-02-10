package buffer

import "testing"

func TestBuffer_SetCursor_ClampsAndVersions(t *testing.T) {
	b := New("a\nbc", Options{})
	if b.Version() != 0 {
		t.Fatalf("expected version 0, got %d", b.Version())
	}

	b.SetCursor(Pos{Row: 999, Col: 999})
	if got := b.Cursor(); got != (Pos{Row: 1, Col: 2}) {
		t.Fatalf("cursor=%v, want (1,2)", got)
	}
	if b.Version() != 1 {
		t.Fatalf("expected version 1, got %d", b.Version())
	}

	b.SetCursor(Pos{Row: 1, Col: 2})
	if b.Version() != 1 {
		t.Fatalf("expected version unchanged, got %d", b.Version())
	}
}

func TestBuffer_SetSelection_NormalizesClampsAndVersions(t *testing.T) {
	b := New("a\nbc", Options{})

	b.SetSelection(Range{
		Start: Pos{Row: 1, Col: 99},
		End:   Pos{Row: 0, Col: -1},
	})

	r, ok := b.Selection()
	if !ok {
		t.Fatalf("expected selection active")
	}
	want := Range{Start: Pos{Row: 0, Col: 0}, End: Pos{Row: 1, Col: 2}}
	if r != want {
		t.Fatalf("selection=%v, want %v", r, want)
	}
	if b.Version() != 1 {
		t.Fatalf("expected version 1, got %d", b.Version())
	}

	// Setting the same effective selection should not bump the version.
	b.SetSelection(Range{Start: Pos{Row: 1, Col: 2}, End: Pos{Row: 0, Col: 0}})
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
