package editor

import (
	"testing"

	"github.com/iw2rmb/flouris/buffer"
)

func TestHitTest_NoLineNums_ClampsAndYOffset(t *testing.T) {
	m := New(Config{Text: "abc\ndef\nghi"})
	m.viewport.YOffset = 1

	if got := m.screenToDocPos(2, 0); got != (buffer.Pos{Row: 1, Col: 2}) {
		t.Fatalf("pos at (2,0) with yoffset=1: got %v, want %v", got, buffer.Pos{Row: 1, Col: 2})
	}

	// Clamp x past end of line.
	if got := m.screenToDocPos(999, 0); got != (buffer.Pos{Row: 1, Col: 3}) {
		t.Fatalf("pos at (999,0): got %v, want %v", got, buffer.Pos{Row: 1, Col: 3})
	}
}

func TestHitTest_WithLineNums_GutterMapsToStartOfLine(t *testing.T) {
	m := New(Config{Text: "abcd\nefgh", ShowLineNums: true})

	// 2 lines => 1 digit + 1 gutter space => width 2.
	if got := m.screenToDocPos(0, 0); got != (buffer.Pos{Row: 0, Col: 0}) {
		t.Fatalf("gutter click x=0: got %v, want %v", got, buffer.Pos{Row: 0, Col: 0})
	}
	if got := m.screenToDocPos(1, 0); got != (buffer.Pos{Row: 0, Col: 0}) {
		t.Fatalf("gutter click x=1: got %v, want %v", got, buffer.Pos{Row: 0, Col: 0})
	}

	// First text cell is x=2.
	if got := m.screenToDocPos(2, 0); got != (buffer.Pos{Row: 0, Col: 0}) {
		t.Fatalf("first cell x=2: got %v, want %v", got, buffer.Pos{Row: 0, Col: 0})
	}
	if got := m.screenToDocPos(3, 0); got != (buffer.Pos{Row: 0, Col: 1}) {
		t.Fatalf("second cell x=3: got %v, want %v", got, buffer.Pos{Row: 0, Col: 1})
	}
}

