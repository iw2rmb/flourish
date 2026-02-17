package buffer

import (
	"testing"

	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

func TestBuffer_MoveGrapheme_BoundsAndLineCrossing(t *testing.T) {
	b := New("ab\nçd", Options{})

	b.SetCursor(Pos{Row: 0, GraphemeCol: 0})
	b.Move(Move{Unit: MoveGrapheme, Dir: DirLeft})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor=%v, want (0,0)", got)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor=%v, want (0,1)", got)
	}

	b.SetCursor(Pos{Row: 0, GraphemeCol: 2})
	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 1, GraphemeCol: 0}) {
		t.Fatalf("cursor=%v, want (1,0)", got)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirLeft})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want (0,2)", got)
	}
}

func TestBuffer_MoveLine_HomeEndAndVerticalClamp(t *testing.T) {
	b := New("hello\nw\nworld", Options{})

	b.SetCursor(Pos{Row: 0, GraphemeCol: 3})
	b.Move(Move{Unit: MoveLine, Dir: DirEnd})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("cursor=%v, want (0,5)", got)
	}

	b.Move(Move{Unit: MoveLine, Dir: DirHome})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor=%v, want (0,0)", got)
	}

	b.SetCursor(Pos{Row: 2, GraphemeCol: 5})
	b.Move(Move{Unit: MoveLine, Dir: DirUp})
	if got := b.Cursor(); got != (Pos{Row: 1, GraphemeCol: 1}) {
		t.Fatalf("cursor=%v, want (1,1)", got)
	}
}

func TestBuffer_MoveDoc_StartEnd(t *testing.T) {
	b := New("a\nbc", Options{})

	b.SetCursor(Pos{Row: 1, GraphemeCol: 1})
	b.Move(Move{Unit: MoveDoc, Dir: DirHome})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor=%v, want (0,0)", got)
	}

	b.Move(Move{Unit: MoveDoc, Dir: DirEnd})
	if got := b.Cursor(); got != (Pos{Row: 1, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want (1,2)", got)
	}

	b.Move(Move{Unit: MoveDoc, Dir: DirUp})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor=%v, want (0,0)", got)
	}

	b.Move(Move{Unit: MoveDoc, Dir: DirDown})
	if got := b.Cursor(); got != (Pos{Row: 1, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want (1,2)", got)
	}
}

func TestBuffer_Move_ExtendSelectionAnchorStability(t *testing.T) {
	b := New("abcd", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight, Extend: true})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want (0,2)", got)
	}
	if !b.sel.active || b.sel.anchor != (Pos{Row: 0, GraphemeCol: 1}) || b.sel.end != (Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("sel=%#v, want active anchor(0,1) end(0,2)", b.sel)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight, Extend: true})
	if !b.sel.active || b.sel.anchor != (Pos{Row: 0, GraphemeCol: 1}) || b.sel.end != (Pos{Row: 0, GraphemeCol: 3}) {
		t.Fatalf("sel=%#v, want active anchor(0,1) end(0,3)", b.sel)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirLeft, Extend: false})
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if b.sel.active {
		t.Fatalf("expected internal selection cleared, got %#v", b.sel)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirLeft, Extend: true})
	if !b.sel.active || b.sel.anchor != (Pos{Row: 0, GraphemeCol: 2}) || b.sel.end != (Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("sel=%#v, want active anchor(0,2) end(0,1)", b.sel)
	}
}

func TestBuffer_Move_ExtendClearsWhenReturningToAnchor(t *testing.T) {
	b := New("abcd", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight, Extend: true})
	if !b.sel.active {
		t.Fatalf("expected selection active")
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirLeft, Extend: true})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor=%v, want (0,1)", got)
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared when cursor returns to anchor")
	}
}

func TestBuffer_MoveWord_PortableSemantics(t *testing.T) {
	b := New("  foo, bar", Options{})

	b.SetCursor(Pos{Row: 0, GraphemeCol: 0})
	b.Move(Move{Unit: MoveWord, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("cursor=%v, want (0,6)", got)
	}

	b.Move(Move{Unit: MoveWord, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 10}) {
		t.Fatalf("cursor=%v, want (0,10)", got)
	}

	b.Move(Move{Unit: MoveWord, Dir: DirLeft})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 7}) {
		t.Fatalf("cursor=%v, want (0,7)", got)
	}

	b.Move(Move{Unit: MoveWord, Dir: DirLeft})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want (0,2)", got)
	}
}

func TestBuffer_MoveWord_UnicodeAndNewlineBoundary(t *testing.T) {
	greek := "προβλήμα"
	rest := "テスト"
	line := greek + "  " + rest
	b := New(line+"\nbar", Options{})

	b.SetCursor(Pos{Row: 0, GraphemeCol: 0})
	b.Move(Move{Unit: MoveWord, Dir: DirRight})
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: graphemeutil.Count(greek)}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}

	b.SetCursor(Pos{Row: 0, GraphemeCol: graphemeutil.Count(line)})
	b.Move(Move{Unit: MoveWord, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: graphemeutil.Count(line)}) {
		t.Fatalf("cursor=%v, want unchanged at EOL", got)
	}

	b.SetCursor(Pos{Row: 1, GraphemeCol: 0})
	b.Move(Move{Unit: MoveWord, Dir: DirLeft})
	if got := b.Cursor(); got != (Pos{Row: 1, GraphemeCol: 0}) {
		t.Fatalf("cursor=%v, want unchanged at SOL", got)
	}
}

func TestBuffer_Move_Versioning_NoOpAndSelectionOnlyChanges(t *testing.T) {
	b := New("a", Options{})
	v0 := b.Version()

	b.Move(Move{Unit: MoveGrapheme, Dir: DirLeft})
	if got := b.Version(); got != v0 {
		t.Fatalf("version=%d, want unchanged %d", got, v0)
	}

	b2 := New("ab", Options{})
	b2.SetCursor(Pos{Row: 0, GraphemeCol: 1})
	if got := b2.Version(); got != 1 {
		t.Fatalf("version=%d, want 1 after SetCursor", got)
	}

	b2.Move(Move{Unit: MoveGrapheme, Dir: DirRight, Extend: true})
	if got := b2.Cursor(); got != (Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want (0,2)", got)
	}
	if got := b2.Version(); got != 2 {
		t.Fatalf("version=%d, want 2 after extend-move", got)
	}
	if _, ok := b2.Selection(); !ok {
		t.Fatalf("expected selection active")
	}

	b2.Move(Move{Unit: MoveGrapheme, Dir: DirRight, Extend: false})
	if got := b2.Cursor(); got != (Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor=%v, want unchanged at EOL", got)
	}
	if _, ok := b2.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if got := b2.Version(); got != 3 {
		t.Fatalf("version=%d, want 3 after clearing selection", got)
	}

	b2.Move(Move{Unit: MoveGrapheme, Dir: DirRight, Extend: false})
	if got := b2.Version(); got != 3 {
		t.Fatalf("version=%d, want unchanged 3", got)
	}
}

func TestBuffer_MoveGrapheme_CombiningAndZWJ(t *testing.T) {
	line := "a" + "e\u0301" + "👨‍👩‍👧‍👦" + "b"
	b := New(line, Options{})

	if got, want := b.lineLen(0), 4; got != want {
		t.Fatalf("line grapheme len=%d, want %d", got, want)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("cursor after first right=%v, want (0,1)", got)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor after combining cluster=%v, want (0,2)", got)
	}

	b.Move(Move{Unit: MoveGrapheme, Dir: DirRight})
	if got := b.Cursor(); got != (Pos{Row: 0, GraphemeCol: 3}) {
		t.Fatalf("cursor after ZWJ cluster=%v, want (0,3)", got)
	}
}
