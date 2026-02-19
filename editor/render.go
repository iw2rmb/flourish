package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/iw2rmb/flourish/buffer"
	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

func (m *Model) renderContent() string {
	if m.buf == nil {
		return ""
	}

	lines := rawLinesFromBufferText(m.buf.Text())
	layout := m.ensureLayoutCache(lines)

	cursor := m.buf.Cursor()
	sel, selOK := m.buf.Selection()
	digitCount := 0
	if m.cfg.ShowLineNums {
		digitCount = gutterDigits(len(lines))
	}

	highlightsByLine := make([][]HighlightSpan, len(layout.lines))
	if m.cfg.Highlighter != nil {
		h := m.viewport.Height - m.viewport.Style.GetVerticalFrameSize()
		if h > 0 {
			start := m.viewport.YOffset
			if start < 0 {
				start = 0
			}
			if start > len(layout.rows) {
				start = len(layout.rows)
			}
			end := start + h
			if end > len(layout.rows) {
				end = len(layout.rows)
			}

			marked := make([]bool, len(layout.lines))
			for visualRow := start; visualRow < end; visualRow++ {
				ref := layout.rows[visualRow]
				row := ref.logicalRow
				if row < 0 || row >= len(layout.lines) || marked[row] {
					continue
				}
				marked[row] = true
				line := layout.lines[row]
				highlightsByLine[row] = m.highlightForLine(row, line.rawLine, line.vt, cursor)
			}
		}
	}

	out := make([]string, 0, len(layout.rows))
	maxIntVal := int(^uint(0) >> 1)
	contentWidth := m.contentWidth(len(lines))
	leftNoWrap := maxInt(m.xOffset, 0)
	rightNoWrap := maxIntVal
	if m.cfg.WrapMode == WrapNone && contentWidth > 0 {
		rightNoWrap = leftNoWrap + contentWidth
	}

	for _, ref := range layout.rows {
		row := ref.logicalRow
		if row < 0 || row >= len(layout.lines) {
			continue
		}
		line := layout.lines[row]
		if ref.segmentIndex < 0 || ref.segmentIndex >= len(line.segments) {
			continue
		}
		seg := line.segments[ref.segmentIndex]

		var sb strings.Builder

		if m.cfg.ShowLineNums {
			numStyle := m.cfg.Style.LineNum
			if m.focused && row == cursor.Row && ref.segmentIndex == 0 {
				numStyle = m.cfg.Style.LineNumActive
			}
			num := fmt.Sprintf("%*s", digitCount, "")
			if ref.segmentIndex == 0 {
				num = fmt.Sprintf("%*d", digitCount, row+1)
			}
			sb.WriteString(numStyle.Render(num))
			sb.WriteString(m.cfg.Style.Gutter.Render(" "))
		}

		left := leftNoWrap
		right := rightNoWrap
		if m.cfg.WrapMode != WrapNone {
			left = seg.startCell
			right = seg.endCell
			if seg.Cells == 0 {
				right = left + 1
			}
		}
		sb.WriteString(renderVisualLine(
			m.cfg.Style,
			m.cfg.GhostStyleForKey,
			m.cfg.VirtualOverlayStyleForKey,
			line.visual,
			row,
			cursor,
			m.focused,
			sel,
			selOK,
			highlightsByLine[row],
			left,
			right,
		))

		out = append(out, sb.String())
	}

	return strings.Join(out, "\n")
}

func (m *Model) highlightForLine(row int, rawLine string, vt VirtualText, cursor buffer.Pos) []HighlightSpan {
	if m.cfg.Highlighter == nil {
		return nil
	}

	visible, rawToVisible := visibleTextAfterDeletions(rawLine, vt)
	visLen := graphemeutil.Count(visible)

	hasCursor := cursor.Row == row
	cursorCol := -1
	rawCursorCol := -1
	if hasCursor {
		rawLen := graphemeutil.Count(rawLine)
		rawCursorCol = clampInt(cursor.GraphemeCol, 0, rawLen)
		if rawCursorCol >= 0 && rawCursorCol < len(rawToVisible) {
			cursorCol = clampInt(rawToVisible[rawCursorCol], 0, visLen)
		} else {
			cursorCol = visLen
		}
	}

	spans, err := m.cfg.Highlighter.HighlightLine(LineContext{
		Row:                  row,
		RawText:              rawLine,
		Text:                 visible,
		CursorGraphemeCol:    cursorCol,
		RawCursorGraphemeCol: rawCursorCol,
		HasCursor:            hasCursor,
	})
	if err != nil {
		return nil
	}
	return normalizeHighlightSpans(spans, visLen)
}

func renderVisualLine(
	st Style,
	ghostStyleForKey func(string) (lipgloss.Style, bool),
	overlayStyleForKey func(string) (lipgloss.Style, bool),
	vl VisualLine,
	row int,
	cursor buffer.Pos,
	focused bool,
	sel buffer.Range,
	selOK bool,
	highlights []HighlightSpan,
	left, right int,
) string {
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
			spanL := maxInt(segL, left)
			spanR := minInt(segR, right)
			if spanL < spanR {
				eolBoundaryCursorTokenIdx = i
				break
			}
		}
	}

	left = maxInt(left, 0)
	if right < left {
		right = left
	}

	isAllSpaces := func(s string) bool {
		if s == "" {
			return false
		}
		for _, g := range graphemeutil.Split(s) {
			if !graphemeutil.IsSpace(g) {
				return false
			}
		}
		return true
	}

	renderSpan := func(styleFn func(...string) string, text string, tokWidth, spanStart, spanWidth int, splittable bool) string {
		if spanWidth <= 0 {
			return ""
		}
		if spanStart == 0 && spanWidth == tokWidth {
			return styleFn(text)
		}
		if splittable {
			return styleFn(strings.Repeat(" ", spanWidth))
		}
		// Partial wide grapheme: preserve alignment with blanks.
		return st.Text.Render(strings.Repeat(" ", spanWidth))
	}

	isTrailingWhitespaceFrom := func(tokIdx int) bool {
		for j := tokIdx; j < len(vl.Tokens); j++ {
			tok := vl.Tokens[j]
			segL := tok.StartCell
			segR := tok.StartCell + tok.CellWidth
			spanL := maxInt(segL, left)
			spanR := minInt(segR, right)
			if spanL >= spanR {
				continue
			}
			if !isAllSpaces(tok.Text) {
				return false
			}
		}
		return true
	}

	var sb strings.Builder
	for i, tok := range vl.Tokens {
		if renderEOLCursor && eolCursorCell == tok.StartCell {
			// Cursor placeholder sits immediately before insertions anchored at EOL.
			spanL := maxInt(eolCursorCell, left)
			spanR := minInt(eolCursorCell+1, right)
			if spanL < spanR {
				sb.WriteString(st.Cursor.Render(" "))
			}
		}

		segL := tok.StartCell
		segR := tok.StartCell + tok.CellWidth
		spanL := maxInt(segL, left)
		spanR := minInt(segR, right)
		if spanL >= spanR {
			continue
		}
		spanStart := spanL - segL
		spanWidth := spanR - spanL
		splittable := isAllSpaces(tok.Text) && tok.CellWidth == graphemeutil.Count(tok.Text)

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
			selected := hasSel && tok.DocStartGraphemeCol < selEndCol && tok.DocEndGraphemeCol > selStartCol
			if hasCursor && (i == cursorTokenIdx || i == eolBoundaryCursorTokenIdx) {
				cursorStyle := st.Cursor.Render
				if isAllSpaces(tok.Text) && isTrailingWhitespaceFrom(i) {
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
				write(renderSpan(cursorStyle, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			} else if selected {
				write(renderSpan(st.Selection.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			} else {
				style := st.Text
				for _, sp := range highlights {
					if tok.VisibleStartGraphemeCol < sp.EndGraphemeCol && tok.VisibleEndGraphemeCol > sp.StartGraphemeCol {
						style = sp.Style.Inherit(st.Text)
						break
					}
				}
				write(renderSpan(style.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			}
		default:
			write(renderSpan(st.Text.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
		}
	}
	if renderEOLCursor && eolCursorCell == vl.VisualLen() {
		spanL := maxInt(eolCursorCell, left)
		spanR := minInt(eolCursorCell+1, right)
		if spanL < spanR {
			sb.WriteString(st.Cursor.Render(" "))
		}
	}
	return sb.String()
}
