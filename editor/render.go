package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

func (m Model) renderScrollbarChrome(base string) string {
	if m.buf == nil {
		return base
	}

	mm := &m
	lines := mm.ensureLines()
	layout := mm.ensureLayoutCache(lines)
	metrics := mm.resolveScrollbarMetrics(lines, layout)
	if !metrics.showV && !metrics.showH {
		return base
	}
	if metrics.innerWidth <= 0 || metrics.innerHeight <= 0 {
		return base
	}

	leftFrame := m.viewport.Style.GetMarginLeft() + m.viewport.Style.GetBorderLeftSize() + m.viewport.Style.GetPaddingLeft()
	topFrame := m.viewport.Style.GetMarginTop() + m.viewport.Style.GetBorderTopSize() + m.viewport.Style.GetPaddingTop()
	view := base

	if metrics.showV && metrics.contentHeight > 0 {
		trackCell := m.cfg.Style.ScrollbarTrack.Render(" ")
		thumbCell := m.cfg.Style.ScrollbarThumb.Render(" ")
		col := renderScrollbarAxisCells(metrics.contentHeight, metrics.vThumbPos, metrics.vThumbLen, trackCell, thumbCell, "\n")
		if col != "" {
			view = compositeTopLeft(
				col,
				view,
				leftFrame+metrics.innerWidth-1,
				topFrame,
			)
		}
	}

	if metrics.showH {
		clearCell := m.cfg.Style.Text.Render(" ")
		row := renderScrollbarAxisCells(metrics.innerWidth, 0, 0, clearCell, clearCell, "")
		if row != "" {
			view = compositeTopLeft(
				row,
				view,
				leftFrame,
				topFrame+metrics.innerHeight-1,
			)
		}

		if metrics.contentWidth > 0 {
			trackCell := m.cfg.Style.ScrollbarTrack.Render(" ")
			thumbCell := m.cfg.Style.ScrollbarThumb.Render(" ")
			hRow := renderScrollbarAxisCells(metrics.contentWidth, metrics.hThumbPos, metrics.hThumbLen, trackCell, thumbCell, "")
			if hRow != "" {
				gutterWidth := mm.resolvedGutterWidth(len(lines))
				view = compositeTopLeft(
					hRow,
					view,
					leftFrame+gutterWidth,
					topFrame+metrics.innerHeight-1,
				)
			}
		}
	}

	if metrics.showV && metrics.showH {
		corner := m.cfg.Style.ScrollbarCorner.Render(" ")
		view = compositeTopLeft(
			corner,
			view,
			leftFrame+metrics.innerWidth-1,
			topFrame+metrics.innerHeight-1,
		)
	}

	return view
}

func renderScrollbarAxisCells(length, thumbPos, thumbLen int, trackCell, thumbCell, sep string) string {
	if length <= 0 {
		return ""
	}
	if thumbLen < 0 {
		thumbLen = 0
	}
	thumbStart := clampInt(thumbPos, 0, length)
	thumbEnd := clampInt(thumbPos+thumbLen, 0, length)

	var sb strings.Builder
	for i := 0; i < length; i++ {
		if i > 0 && sep != "" {
			sb.WriteString(sep)
		}
		if i >= thumbStart && i < thumbEnd {
			sb.WriteString(thumbCell)
			continue
		}
		sb.WriteString(trackCell)
	}
	return sb.String()
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
	baseGutterWidth := m.resolvedBaseGutterWidth(lineCount)
	rowMarkWidth := m.resolvedRowMarkWidth()

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
			baseGutterWidth,
			rowMarkWidth,
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
	lineCount, contentWidth, baseGutterWidth, rowMarkWidth int,
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
	needVisibleInfo := m.cfg.RowStyleForRow != nil || m.cfg.TokenStyleForToken != nil || m.cfg.LinkProvider != nil
	if needVisibleInfo && !line.visibleInfoComputed {
		line.visibleInfo = computeVisibleLineInfo(line.rawLine, line.vt)
		line.visibleInfoComputed = true
	}
	if !line.linksResolved {
		line.links = m.linksForLine(row, line.rawLine, line.visibleInfo, cursor)
		line.linksResolved = true
	}
	if ref.segmentIndex < 0 || ref.segmentIndex >= len(line.segments) {
		return "", false
	}
	seg := line.segments[ref.segmentIndex]

	var sb strings.Builder
	if rowMarkWidth > 0 {
		cell := m.resolveRowMarkCell(row, ref.segmentIndex, line.rawLine, rowMarkWidth, row == cursor.Row)
		sb.WriteString(renderGutterCell(m.cfg.Style.Gutter, nil, cell))
	}
	if baseGutterWidth > 0 {
		cell := m.resolveGutterCell(row, ref.segmentIndex, line.rawLine, lineCount, baseGutterWidth, row == cursor.Row)
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
	rowStyle := lipgloss.Style{}
	rowPaintStyle := lipgloss.Style{}
	rowStyleSet := false
	if m.cfg.RowStyleForRow != nil {
		rowText := line.rawLine
		if line.visibleInfoComputed {
			rowText = line.visibleInfo.visible
		}
		style, ok := m.cfg.RowStyleForRow(RowStyleContext{
			Row:          row,
			SegmentIndex: ref.segmentIndex,
			RawText:      line.rawLine,
			Text:         rowText,
			IsActive:     row == cursor.Row,
			IsFocused:    m.focused,
		})
		if ok {
			rowStyle = style
			rowPaintStyle = sanitizeRowPaintStyle(style)
			rowStyleSet = true
		}
	}
	var contentSB strings.Builder
	renderVisualLine(
		&contentSB,
		m.cfg.Style,
		m.cfg.GhostStyleForKey,
		m.cfg.VirtualOverlayStyleForKey,
		m.cfg.TokenStyleForToken,
		rowPaintStyle,
		ref.segmentIndex,
		line.rawLine,
		line.visibleInfo.visible,
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
	content := contentSB.String()
	if rowStyleSet && contentWidth > 0 {
		base := fitRenderedRowWidth(content, contentWidth, rowPaintStyle.Inherit(m.cfg.Style.Text))
		content = fitRenderedRowWidth(renderRowBoxStyle(base, rowStyle, contentWidth), contentWidth, rowPaintStyle.Inherit(m.cfg.Style.Text))
	}
	sb.WriteString(content)
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
	tokenStyleForToken func(TokenStyleContext) (lipgloss.Style, bool),
	rowPaintStyle lipgloss.Style,
	segmentIndex int,
	rawText string,
	text string,
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
	rowBaseStyle := rowPaintStyle.Inherit(st.Text)

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
		return rowBaseStyle.Render(spaceString(spanWidth))
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

	applyTokenStyle := func(
		style lipgloss.Style,
		tok VisualToken,
		isHighlighted bool,
		isSelected bool,
		isLink bool,
		linkTarget string,
	) lipgloss.Style {
		if tokenStyleForToken == nil {
			return style
		}
		if callbackStyle, ok := tokenStyleForToken(TokenStyleContext{
			Row:           row,
			SegmentIndex:  segmentIndex,
			Token:         tok,
			RawText:       rawText,
			Text:          text,
			IsActiveRow:   row == cursor.Row,
			IsFocused:     focused,
			IsHighlighted: isHighlighted,
			IsSelected:    isSelected,
			IsLink:        isLink,
			LinkTarget:    linkTarget,
		}); ok {
			return callbackStyle.Inherit(style)
		}
		return style
	}

	linkIdx := 0 // advancing pointer into sorted links slice
	for i, tok := range vl.Tokens {
		if renderEOLCursor && eolCursorCell == tok.StartCell {
			// Cursor placeholder sits immediately before insertions anchored at EOL.
			spanL := max(eolCursorCell, left)
			spanR := min(eolCursorCell+1, right)
			if spanL < spanR {
				sb.WriteString(st.Cursor.Inherit(rowBaseStyle).Render(" "))
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
			style := rowBaseStyle
			switch tok.Role {
			case VirtualRoleGhost:
				style = st.Ghost.Inherit(rowBaseStyle)
				if ghostStyleForKey != nil && tok.StyleKey != "" {
					if keyed, ok := ghostStyleForKey(tok.StyleKey); ok {
						style = keyed.Inherit(rowBaseStyle)
					}
				}
			case VirtualRoleOverlay:
				style = st.VirtualOverlay.Inherit(rowBaseStyle)
				if overlayStyleForKey != nil && tok.StyleKey != "" {
					if keyed, ok := overlayStyleForKey(tok.StyleKey); ok {
						style = keyed.Inherit(rowBaseStyle)
					}
				}
			}
			style = applyTokenStyle(style, tok, false, false, false, "")
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
			highlighted := false
			if len(highlights) > 0 {
				for _, sp := range highlights {
					if tok.VisibleStartGraphemeCol < sp.EndGraphemeCol && tok.VisibleEndGraphemeCol > sp.StartGraphemeCol {
						highlighted = true
						break
					}
				}
			}
			if hasCursor && (i == cursorTokenIdx || i == eolBoundaryCursorTokenIdx) {
				cursorStyleDef := st.Cursor.Inherit(rowBaseStyle)
				if tok.AllSpaces && isTrailingWhitespaceFrom(i) {
					// Trailing ASCII spaces can be visually elided by terminals at line end.
					// Render cursor whitespace as NBSP in that case so the cursor stays visible.
					cursorStyle := func(parts ...string) string {
						replaced := make([]string, len(parts))
						for i, p := range parts {
							replaced[i] = strings.ReplaceAll(p, " ", "\u00a0")
						}
						return cursorStyleDef.Render(replaced...)
					}
					writeDoc(renderSpan(cursorStyle, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
					continue
				}
				writeDoc(renderSpan(cursorStyleDef.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			} else {
				style := rowBaseStyle
				for _, sp := range highlights {
					if tok.VisibleStartGraphemeCol < sp.EndGraphemeCol && tok.VisibleEndGraphemeCol > sp.StartGraphemeCol {
						style = sp.Style.Inherit(style)
					}
				}
				if linkTarget != "" {
					style = linkStyle.Inherit(style)
				}
				style = applyTokenStyle(style, tok, highlighted, selected, linkTarget != "", linkTarget)
				if selected {
					style = st.Selection.Inherit(style)
				}
				writeDoc(renderSpan(style.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			}
		default:
			style := rowBaseStyle
			style = applyTokenStyle(style, tok, false, false, false, "")
			write(renderSpan(style.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
		}
	}
	if renderEOLCursor && eolCursorCell == vl.VisualLen() {
		spanL := max(eolCursorCell, left)
		spanR := min(eolCursorCell+1, right)
		if spanL < spanR {
			sb.WriteString(st.Cursor.Inherit(rowBaseStyle).Render(" "))
		}
	}
}

func sanitizeRowPaintStyle(s lipgloss.Style) lipgloss.Style {
	return s.
		UnsetMargins().
		UnsetPadding().
		UnsetWidth().
		UnsetHeight().
		UnsetMaxWidth().
		UnsetMaxHeight().
		UnsetAlign().
		UnsetAlignHorizontal().
		UnsetAlignVertical().
		UnsetInline().
		UnsetTransform().
		UnsetBorderStyle().
		UnsetBorderTop().
		UnsetBorderRight().
		UnsetBorderBottom().
		UnsetBorderLeft()
}

func firstLineOnly(s string) string {
	if s == "" {
		return ""
	}
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func fitRenderedRowWidth(s string, width int, fillStyle lipgloss.Style) string {
	if width <= 0 {
		return s
	}
	s = firstLineOnly(s)
	s = ansi.Truncate(s, width, "")
	if w := ansi.StringWidth(s); w < width {
		s += fillStyle.Render(spaceString(width - w))
	}
	return s
}

func renderRowBoxStyle(base string, boxStyle lipgloss.Style, width int) string {
	if width <= 0 {
		return firstLineOnly(boxStyle.Render(base))
	}
	boxed := firstLineOnly(boxStyle.Render(base))
	boxed = ansi.Truncate(boxed, width, "")
	if w := ansi.StringWidth(boxed); w < width {
		boxed += ansi.Cut(base, w, width)
	}
	return boxed
}
