package editor

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iw2rmb/flourish/buffer"
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
		Text:   "one\ntwo\nthree\nfour\nfive",
		Gutter: LineNumberGutter(),
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

	m.buf.SetCursor(buffer.Pos{Row: 1, GraphemeCol: 1})
	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 1},
		End:   buffer.Pos{Row: 2, GraphemeCol: 1},
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

	if !got[1].HasCursor || got[1].CursorGraphemeCol != 1 {
		t.Fatalf("cursor ctx row 1: got (has=%v,col=%d), want (true,1)", got[1].HasCursor, got[1].CursorGraphemeCol)
	}
	if got[0].HasCursor || got[2].HasCursor {
		t.Fatalf("cursor ctx other rows: got row0=%v row2=%v, want both false", got[0].HasCursor, got[2].HasCursor)
	}

	if !got[0].HasSelection || got[0].SelectionStartGraphemeCol != 1 || got[0].SelectionEndGraphemeCol != 2 {
		t.Fatalf("selection ctx row 0: got (has=%v,%d..%d), want (true,1..2)", got[0].HasSelection, got[0].SelectionStartGraphemeCol, got[0].SelectionEndGraphemeCol)
	}
	if !got[1].HasSelection || got[1].SelectionStartGraphemeCol != 0 || got[1].SelectionEndGraphemeCol != 2 {
		t.Fatalf("selection ctx row 1: got (has=%v,%d..%d), want (true,0..2)", got[1].HasSelection, got[1].SelectionStartGraphemeCol, got[1].SelectionEndGraphemeCol)
	}
	if !got[2].HasSelection || got[2].SelectionStartGraphemeCol != 0 || got[2].SelectionEndGraphemeCol != 1 {
		t.Fatalf("selection ctx row 2: got (has=%v,%d..%d), want (true,0..1)", got[2].HasSelection, got[2].SelectionStartGraphemeCol, got[2].SelectionEndGraphemeCol)
	}
}

func TestModel_InvalidateGutter_RebuildsFromExternalState(t *testing.T) {
	label := "A"
	m := New(Config{
		Text: "x",
		Gutter: Gutter{
			Width: func(GutterWidthContext) int { return 2 },
			Cell: func(GutterCellContext) GutterCell {
				return GutterCell{
					Segments: []GutterSegment{{Text: label}},
				}
			},
		},
	})
	m = m.Blur()
	m = m.SetSize(4, 1)

	if got := strings.TrimRight(stripANSI(m.View()), " "); got != "A x" {
		t.Fatalf("initial gutter render: got %q, want %q", got, "A x")
	}

	label = "B"
	m = m.InvalidateGutter()
	if got := strings.TrimRight(stripANSI(m.View()), " "); got != "B x" {
		t.Fatalf("gutter render after invalidate: got %q, want %q", got, "B x")
	}
}

func TestModel_InvalidateGutterRows_EmptyIsNoop(t *testing.T) {
	m := New(Config{
		Text:   "x",
		Gutter: LineNumberGutter(),
	})
	m = m.Blur()
	m = m.SetSize(4, 1)
	before := m.gutterInvalidationVersion

	m = m.InvalidateGutterRows()
	if got := m.gutterInvalidationVersion; got != before {
		t.Fatalf("empty row invalidation should be no-op: got version %d, want %d", got, before)
	}
}

func TestModel_InvalidateGutterRows_RerendersOnlyTargetRows(t *testing.T) {
	callsByRow := map[int]int{}
	m := New(Config{
		Text: "a\nb\nc",
		Gutter: Gutter{
			Width: func(GutterWidthContext) int { return 2 },
			Cell: func(ctx GutterCellContext) GutterCell {
				callsByRow[ctx.Row]++
				return GutterCell{
					Segments: []GutterSegment{{Text: fmt.Sprintf("%d", ctx.Row)}},
				}
			},
		},
	})
	m = m.Blur()
	m = m.SetSize(6, 3)

	callsByRow = map[int]int{}
	m = m.InvalidateGutterRows(1)

	if got := callsByRow[0]; got != 0 {
		t.Fatalf("row 0 gutter rerender calls: got %d, want %d", got, 0)
	}
	if got := callsByRow[1]; got != 1 {
		t.Fatalf("row 1 gutter rerender calls: got %d, want %d", got, 1)
	}
	if got := callsByRow[2]; got != 0 {
		t.Fatalf("row 2 gutter rerender calls: got %d, want %d", got, 0)
	}
}

func TestUpdate_CursorMove_RecomputesVirtualTextOnlyForDirtyRows(t *testing.T) {
	const lineCount = 200
	line := "abcdef"
	text := strings.Repeat(line+"\n", lineCount-1) + line

	callsByRow := map[int]int{}
	m := New(Config{
		Text: text,
		VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
			callsByRow[ctx.Row]++
			return VirtualText{}
		},
	})
	m = m.SetSize(80, 10)

	callsByRow = map[int]int{}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})

	if len(callsByRow) == 0 {
		t.Fatalf("expected virtual text provider call on cursor move")
	}
	if len(callsByRow) > 2 {
		t.Fatalf("cursor move should not rerender all rows: got calls on %d rows, want <=2", len(callsByRow))
	}
	for row := range callsByRow {
		if row != 0 {
			t.Fatalf("unexpected rerendered row %d for single-row cursor move", row)
		}
	}
}
