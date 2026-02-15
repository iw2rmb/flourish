package editor

import "github.com/iw2rmb/flouris/buffer"

// ViewportState is a stable host-facing snapshot of editor camera state.
type ViewportState struct {
	// TopVisualRow is the visual row index rendered at viewport screen row 0.
	TopVisualRow int
	// VisibleRows is the number of content rows available for rendering.
	VisibleRows int
	// LeftCellOffset is the horizontal cell offset in WrapNone mode.
	LeftCellOffset int
	// WrapMode is the active wrapping mode used to interpret coordinates.
	WrapMode WrapMode
}

// ViewportState returns the current host-facing viewport state.
func (m Model) ViewportState() ViewportState {
	top := m.viewport.YOffset
	if top < 0 {
		top = 0
	}

	left := 0
	if m.cfg.WrapMode == WrapNone && m.xOffset > 0 {
		left = m.xOffset
	}

	return ViewportState{
		TopVisualRow:   top,
		VisibleRows:    m.visibleRowCount(),
		LeftCellOffset: left,
		WrapMode:       m.cfg.WrapMode,
	}
}

// ScreenToDoc maps viewport-local screen coordinates to a document position.
//
// Coordinates use terminal cells relative to the editor viewport.
func (m Model) ScreenToDoc(x, y int) buffer.Pos {
	return (&m).screenToDocPos(x, y)
}

// DocToScreen maps a document position to viewport-local screen coordinates.
//
// ok is false when the position is outside the visible viewport content.
func (m Model) DocToScreen(pos buffer.Pos) (x int, y int, ok bool) {
	return (&m).docToScreenPos(pos)
}

func (m Model) visibleRowCount() int {
	h := m.viewport.Height - m.viewport.Style.GetVerticalFrameSize()
	if h < 0 {
		return 0
	}
	return h
}
