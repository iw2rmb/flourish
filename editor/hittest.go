package editor

import (
	"github.com/iw2rmb/flourish/buffer"
)

// screenToDocPos maps viewport-local mouse coordinates to a document position.
//
// Coordinates are in terminal cells and are relative to the editor's viewport:
// (0,0) is the top-left of the visible content region.
//
// v0 mapping rules:
// - gutter clicks map to callback-selected gutter click col (default 0)
// - x/y are clamped into document bounds
func (m *Model) screenToDocPos(x, y int) buffer.Pos {
	if m.buf == nil {
		return buffer.Pos{}
	}

	lines := rawLinesFromBufferText(m.buf.Text())
	layout := m.ensureLayoutCache(lines)
	if len(layout.rows) == 0 {
		return buffer.Pos{}
	}

	visualRow := layout.clampVisualRow(m.viewport.YOffset + y)
	row, line, seg, segIdx, ok := layout.lineAndSegmentAt(visualRow)
	if !ok {
		return buffer.Pos{}
	}

	if x < 0 {
		x = 0
	}
	gw := m.resolvedGutterWidth(len(lines))
	if x < gw {
		cell := m.resolveGutterCell(row, segIdx, line.rawLine, len(lines), gw, row == m.buf.Cursor().Row)
		return buffer.Pos{Row: row, GraphemeCol: clampInt(cell.ClickCol, 0, line.visual.RawGraphemeLen)}
	}
	visualX := x - gw
	if visualX < 0 {
		visualX = 0
	}
	if m.cfg.WrapMode == WrapNone && m.xOffset > 0 {
		visualX += m.xOffset
	}

	if m.cfg.WrapMode != WrapNone {
		if visualX <= 0 {
			return buffer.Pos{Row: row, GraphemeCol: seg.StartGraphemeCol}
		}
		if visualX >= seg.Cells {
			return buffer.Pos{Row: row, GraphemeCol: seg.EndGraphemeCol}
		}
		targetCell := seg.startCell + visualX
		if targetCell >= seg.endCell {
			return buffer.Pos{Row: row, GraphemeCol: seg.EndGraphemeCol}
		}
		col := line.visual.DocGraphemeColForVisualCell(targetCell)
		col = clampInt(col, seg.StartGraphemeCol, seg.EndGraphemeCol)
		return buffer.Pos{Row: row, GraphemeCol: col}
	}

	col := line.visual.DocGraphemeColForVisualCell(visualX)

	return buffer.Pos{Row: row, GraphemeCol: col}
}

// docToScreenPos maps a document position to viewport-local mouse coordinates.
//
// ok is false when the mapped coordinate is outside the visible viewport.
func (m *Model) docToScreenPos(pos buffer.Pos) (x int, y int, ok bool) {
	if m.buf == nil {
		return 0, 0, false
	}

	lines := rawLinesFromBufferText(m.buf.Text())
	layout := m.ensureLayoutCache(lines)
	if len(layout.lines) == 0 || len(layout.rows) == 0 {
		return 0, 0, false
	}

	row := clampInt(pos.Row, 0, len(layout.lines)-1)
	line := layout.lines[row]
	if len(line.segments) == 0 {
		return 0, 0, false
	}

	col := clampInt(pos.GraphemeCol, 0, line.visual.RawGraphemeLen)
	cell := cursorCellForVisualLine(line.visual, col)

	segIdx := len(line.segments) - 1
	for i, seg := range line.segments {
		if seg.Cells == 0 && cell == seg.startCell {
			segIdx = i
			break
		}
		if cell < seg.endCell {
			segIdx = i
			break
		}
	}

	seg := line.segments[segIdx]
	visualRow := line.firstVisualRow + segIdx
	screenY := visualRow - m.viewport.YOffset

	screenX := 0
	if m.cfg.WrapMode == WrapNone {
		screenX = cell - m.xOffset
	} else {
		screenX = cell - seg.startCell
	}
	screenX += m.resolvedGutterWidth(len(lines))

	visibleRows := m.visibleRowCount()
	if screenY < 0 || screenY >= visibleRows {
		return screenX, screenY, false
	}
	if screenX < 0 || screenX >= m.viewport.Width {
		return screenX, screenY, false
	}

	return screenX, screenY, true
}
