package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/iw2rmb/flouris/buffer"
)

func TestModel_SetSizeAffectsViewHeight(t *testing.T) {
	m := New(Config{Text: "a\nb\nc"})
	m = m.Blur()

	m = m.SetSize(20, 2)
	if got := lipgloss.Height(m.View()); got != 2 {
		t.Fatalf("height after SetSize(20,2): got %d, want %d", got, 2)
	}

	m = m.SetSize(20, 4)
	if got := lipgloss.Height(m.View()); got != 4 {
		t.Fatalf("height after SetSize(20,4): got %d, want %d", got, 4)
	}
}

func TestView_SnapshotFixedSize(t *testing.T) {
	m := New(Config{
		Text:         "one\ntwo\nthree\nfour\nfive",
		ShowLineNums: true,
	})
	m = m.Blur()
	m = m.SetSize(8, 3)

	got := strings.Split(m.View(), "\n")
	if len(got) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(got))
	}
	for i := range got {
		got[i] = strings.TrimRight(stripANSI(got[i]), " ")
	}

	want := []string{
		"1 one",
		"2 two",
		"3 three",
	}
	if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
		t.Fatalf("unexpected view:\n got: %q\nwant: %q", got, want)
	}
}

func TestVirtualTextProvider_ContextPerLine(t *testing.T) {
	var got []VirtualTextContext
	m := New(Config{
		Text: "ab\ncd\nef",
		VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
			got = append(got, ctx)
			return VirtualText{}
		},
	})
	got = nil // New() triggers an initial render

	m.buf.SetCursor(buffer.Pos{Row: 1, Col: 1})
	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, Col: 1},
		End:   buffer.Pos{Row: 2, Col: 1},
	})

	_ = m.renderContent()

	if len(got) != 3 {
		t.Fatalf("provider calls: got %d, want %d", len(got), 3)
	}

	if got[0].Row != 0 || got[0].LineText != "ab" {
		t.Fatalf("row 0 ctx: got (%d,%q)", got[0].Row, got[0].LineText)
	}
	if got[1].Row != 1 || got[1].LineText != "cd" {
		t.Fatalf("row 1 ctx: got (%d,%q)", got[1].Row, got[1].LineText)
	}
	if got[2].Row != 2 || got[2].LineText != "ef" {
		t.Fatalf("row 2 ctx: got (%d,%q)", got[2].Row, got[2].LineText)
	}

	if !got[1].HasCursor || got[1].CursorCol != 1 {
		t.Fatalf("cursor ctx row 1: got (has=%v,col=%d), want (true,1)", got[1].HasCursor, got[1].CursorCol)
	}
	if got[0].HasCursor || got[2].HasCursor {
		t.Fatalf("cursor ctx other rows: got row0=%v row2=%v, want both false", got[0].HasCursor, got[2].HasCursor)
	}

	if !got[0].HasSelection || got[0].SelectionStartCol != 1 || got[0].SelectionEndCol != 2 {
		t.Fatalf("selection ctx row 0: got (has=%v,%d..%d), want (true,1..2)", got[0].HasSelection, got[0].SelectionStartCol, got[0].SelectionEndCol)
	}
	if !got[1].HasSelection || got[1].SelectionStartCol != 0 || got[1].SelectionEndCol != 2 {
		t.Fatalf("selection ctx row 1: got (has=%v,%d..%d), want (true,0..2)", got[1].HasSelection, got[1].SelectionStartCol, got[1].SelectionEndCol)
	}
	if !got[2].HasSelection || got[2].SelectionStartCol != 0 || got[2].SelectionEndCol != 1 {
		t.Fatalf("selection ctx row 2: got (has=%v,%d..%d), want (true,0..1)", got[2].HasSelection, got[2].SelectionStartCol, got[2].SelectionEndCol)
	}
}
