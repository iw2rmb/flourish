package editor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

func TestScrollbarMouse_VerticalThumbDrag(t *testing.T) {
	m := New(Config{
		Text: strings.Join([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, "\n"),
		Scrollbar: ScrollbarConfig{
			Horizontal: ScrollbarNever,
		},
	})
	m = m.SetSize(5, 3)

	metrics := scrollbarMouseMetricsForTest(&m)
	if !metrics.showV {
		t.Fatalf("expected vertical scrollbar to be visible")
	}

	x := metrics.innerWidth - 1
	startY := metrics.vThumbPos
	endY := metrics.contentHeight - 1

	m, _ = m.Update(tea.MouseMsg{X: x, Y: startY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m, _ = m.Update(tea.MouseMsg{X: x, Y: endY, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
	m, _ = m.Update(tea.MouseMsg{X: x, Y: endY, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})

	maxY := max(metrics.totalRows-metrics.contentHeight, 0)
	if got := m.ViewportState().TopVisualRow; got != maxY {
		t.Fatalf("top row after vertical thumb drag: got %d, want %d", got, maxY)
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor changed by scrollbar drag: got %v", got)
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("selection must stay clear after scrollbar drag")
	}
}

func TestScrollbarMouse_VerticalTrackClickPages(t *testing.T) {
	m := New(Config{
		Text: strings.Join([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, "\n"),
		Scrollbar: ScrollbarConfig{
			Horizontal: ScrollbarNever,
		},
	})
	m = m.SetSize(5, 3)

	metrics := scrollbarMouseMetricsForTest(&m)
	if !metrics.showV {
		t.Fatalf("expected vertical scrollbar to be visible")
	}
	x := metrics.innerWidth - 1
	downY := metrics.vThumbPos + metrics.vThumbLen
	if downY >= metrics.contentHeight {
		downY = metrics.contentHeight - 1
	}

	m, _ = m.Update(tea.MouseMsg{X: x, Y: downY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got, want := m.ViewportState().TopVisualRow, metrics.contentHeight; got != want {
		t.Fatalf("top row after paging down: got %d, want %d", got, want)
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor changed by vertical scrollbar page-down click: got %v", got)
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("selection must stay clear after vertical scrollbar page-down click")
	}

	metrics = scrollbarMouseMetricsForTest(&m)
	upY := metrics.vThumbPos - 1
	if upY < 0 {
		upY = 0
	}
	m, _ = m.Update(tea.MouseMsg{X: x, Y: upY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got, want := m.ViewportState().TopVisualRow, 0; got != want {
		t.Fatalf("top row after paging up: got %d, want %d", got, want)
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor changed by vertical scrollbar page-up click: got %v", got)
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("selection must stay clear after vertical scrollbar page-up click")
	}
}

func TestScrollbarMouse_HorizontalThumbDrag(t *testing.T) {
	m := New(Config{
		Text: "abcdefghijklmno",
		Scrollbar: ScrollbarConfig{
			Vertical: ScrollbarNever,
		},
	})
	m = m.SetSize(6, 2)

	metrics := scrollbarMouseMetricsForTest(&m)
	if !metrics.showH {
		t.Fatalf("expected horizontal scrollbar to be visible")
	}

	y := metrics.innerHeight - 1
	startX := metrics.hThumbPos
	endX := metrics.contentWidth - 1

	m, _ = m.Update(tea.MouseMsg{X: startX, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m, _ = m.Update(tea.MouseMsg{X: endX, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
	m, _ = m.Update(tea.MouseMsg{X: endX, Y: y, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})

	maxX := max(metrics.totalCols-metrics.contentWidth, 0)
	if got := m.ViewportState().LeftCellOffset; got != maxX {
		t.Fatalf("left offset after horizontal thumb drag: got %d, want %d", got, maxX)
	}
}

func TestScrollbarMouse_HorizontalTrackClickPages(t *testing.T) {
	m := New(Config{
		Text: "abcdefghijklmno",
		Scrollbar: ScrollbarConfig{
			Vertical: ScrollbarNever,
		},
	})
	m = m.SetSize(6, 2)

	metrics := scrollbarMouseMetricsForTest(&m)
	if !metrics.showH {
		t.Fatalf("expected horizontal scrollbar to be visible")
	}
	y := metrics.innerHeight - 1
	rightX := metrics.hThumbPos + metrics.hThumbLen
	if rightX >= metrics.contentWidth {
		rightX = metrics.contentWidth - 1
	}

	m, _ = m.Update(tea.MouseMsg{X: rightX, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got, want := m.ViewportState().LeftCellOffset, metrics.contentWidth; got != want {
		t.Fatalf("left offset after paging right: got %d, want %d", got, want)
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor changed by horizontal scrollbar page-right click: got %v", got)
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("selection must stay clear after horizontal scrollbar page-right click")
	}

	metrics = scrollbarMouseMetricsForTest(&m)
	leftX := metrics.hThumbPos - 1
	if leftX < 0 {
		leftX = 0
	}
	m, _ = m.Update(tea.MouseMsg{X: leftX, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got, want := m.ViewportState().LeftCellOffset, 0; got != want {
		t.Fatalf("left offset after paging left: got %d, want %d", got, want)
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 0}) {
		t.Fatalf("cursor changed by horizontal scrollbar page-left click: got %v", got)
	}
	if _, ok := m.buf.Selection(); ok {
		t.Fatalf("selection must stay clear after horizontal scrollbar page-left click")
	}
}

func TestScrollbarMouse_ScrollFollowCursorOnly_BlocksScrollbarManualActions(t *testing.T) {
	m := New(Config{
		Text:         strings.Join([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, "\n"),
		ScrollPolicy: ScrollFollowCursorOnly,
		Scrollbar: ScrollbarConfig{
			Horizontal: ScrollbarNever,
		},
	})
	m = m.SetSize(5, 3)

	metrics := scrollbarMouseMetricsForTest(&m)
	if !metrics.showV {
		t.Fatalf("expected vertical scrollbar to be visible")
	}
	x := metrics.innerWidth - 1

	m, _ = m.Update(tea.MouseMsg{X: x, Y: metrics.contentHeight - 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got, want := m.ViewportState().TopVisualRow, 0; got != want {
		t.Fatalf("top row after blocked scrollbar press: got %d, want %d", got, want)
	}

	m, _ = m.Update(tea.MouseMsg{X: x, Y: metrics.vThumbPos, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m, _ = m.Update(tea.MouseMsg{X: x, Y: metrics.contentHeight - 1, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
	m, _ = m.Update(tea.MouseMsg{X: x, Y: metrics.contentHeight - 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	if got, want := m.ViewportState().TopVisualRow, 0; got != want {
		t.Fatalf("top row after blocked scrollbar drag: got %d, want %d", got, want)
	}
}

func scrollbarMouseMetricsForTest(m *Model) scrollbarMetrics {
	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	return m.resolveScrollbarMetrics(lines, layout)
}
