package editor

import "github.com/iw2rmb/flouris/buffer"

type wrapLayoutCacheKey struct {
	bufVersion uint64
	cursor     buffer.Pos

	sel   buffer.Range
	selOK bool

	wrapMode     WrapMode
	tabWidth     int
	contentWidth int
	focused      bool
}

type wrapLayoutRow struct {
	logicalRow   int
	segmentIndex int
}

type wrapLayoutLine struct {
	rawLine string
	vt      VirtualText
	visual  VisualLine

	segments       []wrappedSegment
	firstVisualRow int
}

type wrapLayoutCache struct {
	valid bool
	key   wrapLayoutCacheKey

	lines []wrapLayoutLine
	rows  []wrapLayoutRow
}

func (m *Model) invalidateLayoutCache() {
	m.layout.valid = false
}

func (m *Model) layoutKey(lines []string) wrapLayoutCacheKey {
	key := wrapLayoutCacheKey{
		wrapMode:     m.cfg.WrapMode,
		tabWidth:     m.cfg.TabWidth,
		contentWidth: m.contentWidth(len(lines)),
		focused:      m.focused,
	}
	if m.buf == nil {
		return key
	}

	key.bufVersion = m.buf.Version()
	key.cursor = m.buf.Cursor()
	key.sel, key.selOK = m.buf.Selection()
	return key
}

func (m *Model) ensureLayoutCache(lines []string) wrapLayoutCache {
	key := m.layoutKey(lines)
	if m.layout.valid && m.layout.key == key {
		return m.layout
	}

	cache := wrapLayoutCache{
		valid: true,
		key:   key,
		lines: make([]wrapLayoutLine, 0, len(lines)),
		rows:  make([]wrapLayoutRow, 0, len(lines)),
	}

	for row, rawLine := range lines {
		vt := m.virtualTextForRow(row, rawLine)
		vt = m.virtualTextWithGhost(row, rawLine, vt)
		visual := BuildVisualLine(rawLine, vt, m.cfg.TabWidth)
		segments := wrapSegmentsForVisualLine(visual, m.cfg.WrapMode, key.contentWidth)
		if len(segments) == 0 {
			segments = []wrappedSegment{{
				StartCol:  0,
				EndCol:    visual.RawLen,
				Cells:     visual.VisualLen(),
				startCell: 0,
				endCell:   visual.VisualLen(),
			}}
		}

		firstVisualRow := len(cache.rows)
		cache.lines = append(cache.lines, wrapLayoutLine{
			rawLine:        rawLine,
			vt:             vt,
			visual:         visual,
			segments:       segments,
			firstVisualRow: firstVisualRow,
		})
		for segIdx := range segments {
			cache.rows = append(cache.rows, wrapLayoutRow{
				logicalRow:   row,
				segmentIndex: segIdx,
			})
		}
	}

	// Keep a stable zero state when the document is unexpectedly empty.
	if len(cache.lines) == 0 {
		cache.lines = append(cache.lines, wrapLayoutLine{
			rawLine: "",
			visual:  BuildVisualLine("", VirtualText{}, m.cfg.TabWidth),
			segments: []wrappedSegment{{
				StartCol: 0,
				EndCol:   0,
				Cells:    0,
			}},
		})
		cache.rows = append(cache.rows, wrapLayoutRow{})
	}

	m.layout = cache
	return cache
}

func (c wrapLayoutCache) clampVisualRow(row int) int {
	if len(c.rows) == 0 {
		return 0
	}
	return clampInt(row, 0, len(c.rows)-1)
}

func (c wrapLayoutCache) rowAt(visualRow int) (wrapLayoutRow, bool) {
	if len(c.rows) == 0 {
		return wrapLayoutRow{}, false
	}
	visualRow = c.clampVisualRow(visualRow)
	return c.rows[visualRow], true
}

func (c wrapLayoutCache) lineAndSegmentAt(visualRow int) (lineIdx int, line wrapLayoutLine, seg wrappedSegment, segIdx int, ok bool) {
	ref, ok := c.rowAt(visualRow)
	if !ok {
		return 0, wrapLayoutLine{}, wrappedSegment{}, 0, false
	}
	if ref.logicalRow < 0 || ref.logicalRow >= len(c.lines) {
		return 0, wrapLayoutLine{}, wrappedSegment{}, 0, false
	}

	line = c.lines[ref.logicalRow]
	if ref.segmentIndex < 0 || ref.segmentIndex >= len(line.segments) {
		return 0, wrapLayoutLine{}, wrappedSegment{}, 0, false
	}
	return ref.logicalRow, line, line.segments[ref.segmentIndex], ref.segmentIndex, true
}

func (c wrapLayoutCache) cursorVisualPosition(cursor buffer.Pos) (visualRow int, visualCol int, ok bool) {
	if len(c.lines) == 0 {
		return 0, 0, false
	}

	lineIdx := clampInt(cursor.Row, 0, len(c.lines)-1)
	line := c.lines[lineIdx]
	if len(line.segments) == 0 {
		return line.firstVisualRow, 0, true
	}

	cursorCol := clampInt(cursor.Col, 0, line.visual.RawLen)
	cursorCell := cursorCellForVisualLine(line.visual, cursorCol)
	segIdx := len(line.segments) - 1
	for i, seg := range line.segments {
		if seg.Cells == 0 && cursorCell == seg.startCell {
			segIdx = i
			break
		}
		if cursorCell < seg.endCell {
			segIdx = i
			break
		}
	}

	seg := line.segments[segIdx]
	col := cursorCell - seg.startCell
	if col < 0 {
		col = 0
	}
	if seg.Cells == 0 {
		col = 0
	}
	return line.firstVisualRow + segIdx, col, true
}
