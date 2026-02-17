package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

func TestViewportState_ExposesOffsets(t *testing.T) {
	m := New(Config{Text: "0\n1\n2\n3"})
	m = m.SetSize(10, 2)

	st := m.ViewportState()
	if st.TopVisualRow != 0 || st.VisibleRows != 2 || st.LeftCellOffset != 0 || st.WrapMode != WrapNone {
		t.Fatalf("initial viewport state: got %+v", st)
	}

	m, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	st = m.ViewportState()
	if st.TopVisualRow <= 0 {
		t.Fatalf("top row after manual wheel scroll: got %d, want > 0", st.TopVisualRow)
	}
}

func TestViewportState_ScrollFollowCursorOnly_IgnoresManualWheel(t *testing.T) {
	m := New(Config{
		Text:         "0\n1\n2\n3",
		ScrollPolicy: ScrollFollowCursorOnly,
	})
	m = m.SetSize(10, 2)

	m, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	if got := m.ViewportState().TopVisualRow; got != 0 {
		t.Fatalf("top row after manual wheel in follow-cursor mode: got %d, want %d", got, 0)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := m.ViewportState().TopVisualRow; got != 1 {
		t.Fatalf("top row after cursor-driven movement: got %d, want %d", got, 1)
	}
}

func TestDocScreenMapping_UsesViewportOffsets(t *testing.T) {
	m := New(Config{Text: "ab\ncd\nef"})
	m = m.SetSize(10, 2)
	m, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	top := m.ViewportState().TopVisualRow

	if got := m.ScreenToDoc(1, 0); got != (buffer.Pos{Row: top, GraphemeCol: 1}) {
		t.Fatalf("ScreenToDoc at scrolled top: got %v, want %v", got, buffer.Pos{Row: top, GraphemeCol: 1})
	}

	x, y, ok := m.DocToScreen(buffer.Pos{Row: top, GraphemeCol: 1})
	if !ok || x != 1 || y != 0 {
		t.Fatalf("DocToScreen visible pos: got (x=%d,y=%d,ok=%v), want (1,0,true)", x, y, ok)
	}

	if top > 0 {
		_, y, ok = m.DocToScreen(buffer.Pos{Row: top - 1, GraphemeCol: 1})
		if ok || y != -1 {
			t.Fatalf("DocToScreen offscreen row above view: got (y=%d,ok=%v), want (-1,false)", y, ok)
		}
	}
}

func TestDocToScreen_WrapNone_UsesHorizontalOffset(t *testing.T) {
	m := New(Config{Text: "abcdef"})
	m = m.SetSize(3, 1)
	for i := 0; i < 5; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	}

	if got := m.ViewportState().LeftCellOffset; got != 3 {
		t.Fatalf("left offset after cursor move: got %d, want %d", got, 3)
	}

	x, y, ok := m.DocToScreen(buffer.Pos{Row: 0, GraphemeCol: 5})
	if !ok || x != 2 || y != 0 {
		t.Fatalf("DocToScreen cursor col: got (x=%d,y=%d,ok=%v), want (2,0,true)", x, y, ok)
	}

	x, y, ok = m.DocToScreen(buffer.Pos{Row: 0, GraphemeCol: 0})
	if ok || x != -3 || y != 0 {
		t.Fatalf("DocToScreen left-clipped col: got (x=%d,y=%d,ok=%v), want (-3,0,false)", x, y, ok)
	}
}
