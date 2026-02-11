package editor

import (
	"testing"

	"github.com/iw2rmb/flouris/buffer"
)

func TestHitTest_NoLineNums_ClampsAndYOffset(t *testing.T) {
	m := New(Config{Text: "abc\ndef\nghi"})
	m.viewport.YOffset = 1

	if got := m.screenToDocPos(2, 0); got != (buffer.Pos{Row: 1, GraphemeCol: 2}) {
		t.Fatalf("pos at (2,0) with yoffset=1: got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 2})
	}

	// Clamp x past end of line.
	if got := m.screenToDocPos(999, 0); got != (buffer.Pos{Row: 1, GraphemeCol: 3}) {
		t.Fatalf("pos at (999,0): got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 3})
	}
}

func TestHitTest_WithLineNums_GutterMapsToStartOfLine(t *testing.T) {
	m := New(Config{Text: "abcd\nefgh", ShowLineNums: true})

	// 2 lines => 1 digit + 1 gutter space => width 2.
	if got := m.screenToDocPos(0, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("gutter click x=0: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}
	if got := m.screenToDocPos(1, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("gutter click x=1: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}

	// First text cell is x=2.
	if got := m.screenToDocPos(2, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("first cell x=2: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 0})
	}
	if got := m.screenToDocPos(3, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("second cell x=3: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}
}

func TestHitTest_InsertedTextMapsToAnchorCol(t *testing.T) {
	m := New(Config{
		Text: "ab",
		VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
			return VirtualText{Insertions: []VirtualInsertion{{GraphemeCol: 1, Text: "XX"}}}
		},
	})

	// Visual: "aXXb"
	if got := m.screenToDocPos(1, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("click in insertion x=1: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}
	if got := m.screenToDocPos(2, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("click in insertion x=2: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}
}

func TestHitTest_DeletedPrefixMapsFirstVisibleCellToCorrectRawCol(t *testing.T) {
	m := New(Config{
		Text: "**a**",
		VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
			return VirtualText{
				Deletions: []VirtualDeletion{
					{StartGraphemeCol: 0, EndGraphemeCol: 2},
					{StartGraphemeCol: 3, EndGraphemeCol: 5},
				},
			}
		},
	})

	// Visual: "a" at raw col 2.
	if got := m.screenToDocPos(0, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("click first cell x=0: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}

	// Past EOL should clamp to raw EOL (col 5), not the visible EOL (col 3).
	if got := m.screenToDocPos(999, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("click past eol x=999: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}
}

func TestHitTest_TabExpansionUsesCellCoordinates(t *testing.T) {
	m := New(Config{
		Text:     "a\tb",
		TabWidth: 4,
	})

	// Visual cells: "a" [0], tab spaces [1..3], "b" [4].
	if got := m.screenToDocPos(2, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 1}) {
		t.Fatalf("click inside tab x=2: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 1})
	}
	if got := m.screenToDocPos(4, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("click on b x=4: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
}

func TestHitTest_HorizontalScroll_AddsXOffset(t *testing.T) {
	m := New(Config{Text: "abcdef"})
	m.xOffset = 2

	if got := m.screenToDocPos(0, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("click at left edge with xOffset=2: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
	if got := m.screenToDocPos(3, 0); got != (buffer.Pos{Row: 0, GraphemeCol: 5}) {
		t.Fatalf("click x=3 with xOffset=2: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 5})
	}
}

func TestHitTest_SoftWrap_UsesWrappedSegmentsAndViewportYOffset(t *testing.T) {
	m := New(Config{
		Text:     "abcdef\nghijkl",
		WrapMode: WrapGrapheme,
	})
	m = m.SetSize(3, 2)

	// First logical line wraps as: "abc" + "def".
	if got := m.screenToDocPos(1, 1); got != (buffer.Pos{Row: 0, GraphemeCol: 4}) {
		t.Fatalf("wrapped second segment x=1,y=1: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 4})
	}

	// Click past end-of-segment should map to segment end.
	if got := m.screenToDocPos(99, 1); got != (buffer.Pos{Row: 0, GraphemeCol: 6}) {
		t.Fatalf("past wrapped segment end x=99,y=1: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 6})
	}

	// Scroll by visual rows: Y offset 2 starts at line 1, first wrapped segment.
	m.viewport.YOffset = 2
	if got := m.screenToDocPos(1, 0); got != (buffer.Pos{Row: 1, GraphemeCol: 1}) {
		t.Fatalf("wrapped yoffset mapping x=1,y=0,yoffset=2: got %v, want %v", got, buffer.Pos{Row: 1, GraphemeCol: 1})
	}
}
