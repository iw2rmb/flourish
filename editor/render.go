package editor

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/iw2rmb/flourish/buffer"
)

func (m *Model) renderContent() string {
	if m.buf == nil {
		return ""
	}
	m.syncFromBuffer()

	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	metrics := m.resolveScrollbarMetrics(lines, layout)
	rows := m.renderRows(lines, layout, metrics, nil, false)
	return strings.Join(rows, "\n")
}

func (m *Model) renderRows(
	lines []string,
	layout wrapLayoutCache,
	metrics scrollbarMetrics,
	dirtyLogicalRows map[int]struct{},
	useCachedRows bool,
) []string {
	cursor := m.buf.Cursor()
	sel, selOK := m.buf.Selection()
	lineCount := len(lines)
	gutterWidth := m.resolvedGutterWidth(lineCount)

	nLines := len(layout.lines)
	highlightVisible := m.highlightVisible
	highlightsByLine := m.highlightsByLine
	highlightsComputed := m.highlightsComputed
	if cap(highlightVisible) >= nLines {
		highlightVisible = highlightVisible[:nLines]
		highlightsByLine = highlightsByLine[:nLines]
		highlightsComputed = highlightsComputed[:nLines]
	} else {
		highlightVisible = make([]bool, nLines)
		highlightsByLine = make([][]HighlightSpan, nLines)
		highlightsComputed = make([]bool, nLines)
	}
	clear(highlightVisible)
	for i := range highlightsComputed {
		highlightsByLine[i] = nil
		highlightsComputed[i] = false
	}
	m.highlightVisible = highlightVisible
	m.highlightsByLine = highlightsByLine
	m.highlightsComputed = highlightsComputed

	if m.cfg.Highlighter != nil {
		h := metrics.contentHeight
		if h > 0 {
			start := clampInt(metrics.yOffset, 0, len(layout.rows))
			end := start + h
			if end > len(layout.rows) {
				end = len(layout.rows)
			}
			for visualRow := start; visualRow < end; visualRow++ {
				ref := layout.rows[visualRow]
				if ref.logicalRow >= 0 && ref.logicalRow < nLines {
					highlightVisible[ref.logicalRow] = true
				}
			}
		}
	}

	out := make([]string, 0, len(layout.rows))
	maxIntVal := int(^uint(0) >> 1)
	contentWidth := metrics.contentWidth
	leftNoWrap := max(metrics.xOffset, 0)
	rightNoWrap := maxIntVal
	if m.cfg.WrapMode == WrapNone && contentWidth > 0 {
		rightNoWrap = leftNoWrap + contentWidth
	}

	for visualRow, ref := range layout.rows {
		row := ref.logicalRow
		if dirtyLogicalRows != nil {
			if _, dirty := dirtyLogicalRows[row]; !dirty && useCachedRows && visualRow < len(m.renderedRows) {
				out = append(out, m.renderedRows[visualRow])
				continue
			}
		}
		if m.cfg.Highlighter != nil && row >= 0 && row < len(layout.lines) && highlightVisible[row] && !highlightsComputed[row] {
			line := &layout.lines[row]
			if !line.visibleInfoComputed {
				line.visibleInfo = computeVisibleLineInfo(line.rawLine, line.vt)
				line.visibleInfoComputed = true
			}
			highlightsByLine[row] = m.highlightForLine(row, line.rawLine, line.visibleInfo, cursor)
			highlightsComputed[row] = true
		}
		highlights := []HighlightSpan(nil)
		if row >= 0 && row < len(highlightsByLine) {
			highlights = highlightsByLine[row]
		}
		rendered, ok := m.renderLayoutRow(
			layout,
			ref,
			lineCount,
			contentWidth,
			gutterWidth,
			cursor,
			sel,
			selOK,
			highlights,
			leftNoWrap,
			rightNoWrap,
		)
		if !ok {
			if useCachedRows && visualRow < len(m.renderedRows) {
				out = append(out, m.renderedRows[visualRow])
			}
			continue
		}
		out = append(out, rendered)
	}
	return out
}

func (m *Model) renderLayoutRow(
	layout wrapLayoutCache,
	ref wrapLayoutRow,
	lineCount, contentWidth, gutterWidth int,
	cursor buffer.Pos,
	sel buffer.Range,
	selOK bool,
	highlights []HighlightSpan,
	leftNoWrap, rightNoWrap int,
) (string, bool) {
	row := ref.logicalRow
	if row < 0 || row >= len(layout.lines) {
		return "", false
	}
	line := &layout.lines[row]
	if !line.linksResolved {
		if !line.visibleInfoComputed {
			line.visibleInfo = computeVisibleLineInfo(line.rawLine, line.vt)
			line.visibleInfoComputed = true
		}
		line.links = m.linksForLine(row, line.rawLine, line.visibleInfo, cursor)
		line.linksResolved = true
	}
	if ref.segmentIndex < 0 || ref.segmentIndex >= len(line.segments) {
		return "", false
	}
	seg := line.segments[ref.segmentIndex]

	var sb strings.Builder
	if gutterWidth > 0 {
		cell := m.resolveGutterCell(row, ref.segmentIndex, line.rawLine, lineCount, gutterWidth, row == cursor.Row)
		sb.WriteString(renderGutterCell(m.cfg.Style.Gutter, m.cfg.GutterStyleForKey, cell))
	}

	left := leftNoWrap
	right := rightNoWrap
	if m.cfg.WrapMode != WrapNone {
		left = seg.startCell
		right = seg.endCell
		if seg.Cells == 0 {
			right = left + 1
		}
		// Keep EOL cursor one cell past the last glyph on non-full wrapped rows.
		// When the segment already fills content width, fallback rendering in
		// renderVisualLine keeps the cursor visible on the last visible glyph.
		if row == cursor.Row && cursor.GraphemeCol == line.visual.RawGraphemeLen {
			eolCell := cursorCellForVisualLine(line.visual, cursor.GraphemeCol)
			if eolCell == right && seg.Cells < contentWidth {
				right++
			}
		}
	}
	renderVisualLine(
		&sb,
		m.cfg.Style,
		m.cfg.GhostStyleForKey,
		m.cfg.VirtualOverlayStyleForKey,
		line.visual,
		line.links,
		row,
		cursor,
		m.focused,
		sel,
		selOK,
		highlights,
		left,
		right,
	)
	return sb.String(), true
}

func (m *Model) rebuildGutterRows(rows []int) bool {
	if m.buf == nil {
		return false
	}
	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	if len(layout.rows) == 0 || len(m.renderedRows) != len(layout.rows) {
		return false
	}

	dirty := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		if row >= 0 && row < len(layout.lines) {
			dirty[row] = struct{}{}
		}
	}
	if len(dirty) == 0 {
		return true
	}

	metrics := m.resolveScrollbarMetrics(lines, layout)
	rendered := m.renderRows(lines, layout, metrics, dirty, true)
	if len(rendered) != len(layout.rows) {
		return false
	}
	m.setRenderedRows(rendered, metrics)
	return true
}

func (m *Model) highlightForLine(row int, rawLine string, vi visibleLineInfo, cursor buffer.Pos) []HighlightSpan {
	if m.cfg.Highlighter == nil {
		return nil
	}

	hasCursor := cursor.Row == row
	cursorCol := -1
	rawCursorCol := -1
	if hasCursor {
		rawCursorCol = clampInt(cursor.GraphemeCol, 0, vi.rawLen)
		if rawCursorCol >= 0 && rawCursorCol < len(vi.rawToVisible) {
			cursorCol = clampInt(vi.rawToVisible[rawCursorCol], 0, vi.visLen)
		} else {
			cursorCol = vi.visLen
		}
	}

	spans, err := m.cfg.Highlighter.HighlightLine(LineContext{
		Row:                  row,
		RawText:              rawLine,
		Text:                 vi.visible,
		CursorGraphemeCol:    cursorCol,
		RawCursorGraphemeCol: rawCursorCol,
		HasCursor:            hasCursor,
	})
	if err != nil {
		return nil
	}
	return normalizeHighlightSpans(spans, vi.visLen)
}

func renderVisualLine(
	sb *strings.Builder,
	st Style,
	ghostStyleForKey func(string) (lipgloss.Style, bool),
	overlayStyleForKey func(string) (lipgloss.Style, bool),
	vl VisualLine,
	links []resolvedLinkSpan,
	row int,
	cursor buffer.Pos,
	focused bool,
	sel buffer.Range,
	selOK bool,
	highlights []HighlightSpan,
	left, right int,
) {
	rawLen := vl.RawGraphemeLen

	cursorCol := cursor.GraphemeCol
	hasCursor := row == cursor.Row && focused
	if !hasCursor {
		cursorCol = -1
	} else {
		cursorCol = clampInt(cursorCol, 0, rawLen)
	}

	selStartCol, selEndCol, hasSel := selectionColsForRow(sel, selOK, row, rawLen)

	cursorTokenIdx := -1
	if hasCursor && cursorCol >= 0 && cursorCol < rawLen {
		for i, tok := range vl.Tokens {
			if tok.Kind != VisualTokenDoc {
				continue
			}
			if cursorCol >= tok.DocStartGraphemeCol && cursorCol < tok.DocEndGraphemeCol {
				cursorTokenIdx = i
				break
			}
		}
		if cursorTokenIdx == -1 {
			// Cursor is inside a deleted range; snap to the next visible doc-backed token.
			targetCell := vl.VisualCellForDocGraphemeCol(cursorCol)
			for i, tok := range vl.Tokens {
				if tok.Kind == VisualTokenDoc && tok.StartCell == targetCell {
					cursorTokenIdx = i
					break
				}
			}
		}
	}

	// Cursor at EOL is rendered as a 1-cell placeholder space.
	renderEOLCursor := hasCursor && cursorCol == rawLen
	eolCursorCell := -1
	if renderEOLCursor {
		eolCursorCell = cursorCellForVisualLine(vl, cursorCol)
	}
	eolBoundaryCursorTokenIdx := -1
	if renderEOLCursor && eolCursorCell == right && right > left {
		// When wrapped content fully occupies the row, the EOL placeholder cell is
		// outside the visible [left,right) span. Fall back to the last visible
		// doc-backed token so cursor remains visible.
		for i := len(vl.Tokens) - 1; i >= 0; i-- {
			tok := vl.Tokens[i]
			if tok.Kind != VisualTokenDoc {
				continue
			}
			segL := tok.StartCell
			segR := tok.StartCell + tok.CellWidth
			spanL := max(segL, left)
			spanR := min(segR, right)
			if spanL < spanR {
				eolBoundaryCursorTokenIdx = i
				break
			}
		}
	}

	left = max(left, 0)
	if right < left {
		right = left
	}

	renderSpan := func(styleFn func(...string) string, text string, tokWidth, spanStart, spanWidth int, splittable bool) string {
		if spanWidth <= 0 {
			return ""
		}
		if spanStart == 0 && spanWidth == tokWidth {
			return styleFn(text)
		}
		if splittable {
			return styleFn(spaceString(spanWidth))
		}
		// Partial wide grapheme: preserve alignment with blanks.
		return st.Text.Render(spaceString(spanWidth))
	}

	isTrailingWhitespaceFrom := func(tokIdx int) bool {
		for j := tokIdx; j < len(vl.Tokens); j++ {
			tok := vl.Tokens[j]
			segL := tok.StartCell
			segR := tok.StartCell + tok.CellWidth
			spanL := max(segL, left)
			spanR := min(segR, right)
			if spanL >= spanR {
				continue
			}
			if !tok.AllSpaces {
				return false
			}
		}
		return true
	}

	linkIdx := 0 // advancing pointer into sorted links slice
	for i, tok := range vl.Tokens {
		if renderEOLCursor && eolCursorCell == tok.StartCell {
			// Cursor placeholder sits immediately before insertions anchored at EOL.
			spanL := max(eolCursorCell, left)
			spanR := min(eolCursorCell+1, right)
			if spanL < spanR {
				sb.WriteString(st.Cursor.Render(" "))
			}
		}

		segL := tok.StartCell
		segR := tok.StartCell + tok.CellWidth
		spanL := max(segL, left)
		spanR := min(segR, right)
		if spanL >= spanR {
			continue
		}
		spanStart := spanL - segL
		spanWidth := spanR - spanL
		splittable := tok.AllSpaces && tok.CellWidth == tok.GraphemeLen

		write := func(s string) { sb.WriteString(s) }

		switch tok.Kind {
		case VisualTokenVirtual:
			style := st.Text
			switch tok.Role {
			case VirtualRoleGhost:
				style = st.Ghost.Inherit(st.Text)
				if ghostStyleForKey != nil && tok.StyleKey != "" {
					if keyed, ok := ghostStyleForKey(tok.StyleKey); ok {
						style = keyed.Inherit(st.Text)
					}
				}
			case VirtualRoleOverlay:
				style = st.VirtualOverlay.Inherit(st.Text)
				if overlayStyleForKey != nil && tok.StyleKey != "" {
					if keyed, ok := overlayStyleForKey(tok.StyleKey); ok {
						style = keyed.Inherit(st.Text)
					}
				}
			}
			write(renderSpan(style.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
		case VisualTokenDoc:
			// Advance past links that end before this token starts.
			for linkIdx < len(links) && links[linkIdx].EndVisibleGraphemeCol <= tok.VisibleStartGraphemeCol {
				linkIdx++
			}
			linkTarget := ""
			linkStyle := lipgloss.Style{}
			if linkIdx < len(links) {
				link := links[linkIdx]
				if tok.VisibleEndGraphemeCol > link.StartVisibleGraphemeCol {
					linkTarget = link.Target
					linkStyle = link.Style
				}
			}

			writeDoc := func(rendered string) {
				if linkTarget != "" {
					rendered = renderHyperlink(linkTarget, rendered)
				}
				write(rendered)
			}

			selected := hasSel && tok.DocStartGraphemeCol < selEndCol && tok.DocEndGraphemeCol > selStartCol
			if hasCursor && (i == cursorTokenIdx || i == eolBoundaryCursorTokenIdx) {
				cursorStyle := st.Cursor.Render
				if tok.AllSpaces && isTrailingWhitespaceFrom(i) {
					// Trailing ASCII spaces can be visually elided by terminals at line end.
					// Render cursor whitespace as NBSP in that case so the cursor stays visible.
					cursorStyle = func(parts ...string) string {
						replaced := make([]string, len(parts))
						for i, p := range parts {
							replaced[i] = strings.ReplaceAll(p, " ", "\u00a0")
						}
						return st.Cursor.Render(replaced...)
					}
				}
				writeDoc(renderSpan(cursorStyle, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			} else if selected {
				writeDoc(renderSpan(st.Selection.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			} else {
				style := st.Text
				for _, sp := range highlights {
					if tok.VisibleStartGraphemeCol < sp.EndGraphemeCol && tok.VisibleEndGraphemeCol > sp.StartGraphemeCol {
						style = sp.Style.Inherit(style)
					}
				}
				if linkTarget != "" {
					style = linkStyle.Inherit(style)
				}
				writeDoc(renderSpan(style.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			}
		default:
			write(renderSpan(st.Text.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
		}
	}
	if renderEOLCursor && eolCursorCell == vl.VisualLen() {
		spanL := max(eolCursorCell, left)
		spanR := min(eolCursorCell+1, right)
		if spanL < spanR {
			sb.WriteString(st.Cursor.Render(" "))
		}
	}
}
