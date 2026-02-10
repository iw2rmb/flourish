package editor

import (
	"fmt"
	"strings"

	"github.com/iw2rmb/flouris/buffer"
)

func (m *Model) renderContent() string {
	if m.buf == nil {
		return ""
	}

	lines := rawLinesFromBufferText(m.buf.Text())

	cursor := m.buf.Cursor()
	sel, selOK := m.buf.Selection()
	digitCount := 0
	if m.cfg.ShowLineNums {
		digitCount = gutterDigits(len(lines))
	}

	xOffset := 0
	contentWidth := 0
	if m.cfg.WrapMode == WrapNone {
		contentWidth = m.contentWidth(len(lines))
		if contentWidth > 0 {
			xOffset = m.xOffset
		} else {
			contentWidth = 0
		}
	}

	out := make([]string, 0, len(lines))

	highlightStartRow, highlightEndRow := 0, 0
	if m.cfg.Highlighter != nil {
		h := m.viewport.Height - m.viewport.Style.GetVerticalFrameSize()
		if h > 0 {
			highlightStartRow = m.viewport.YOffset
			if highlightStartRow < 0 {
				highlightStartRow = 0
			}
			if highlightStartRow > len(lines) {
				highlightStartRow = len(lines)
			}
			highlightEndRow = highlightStartRow + h
			if highlightEndRow > len(lines) {
				highlightEndRow = len(lines)
			}
		}
	}

	for row, line := range lines {
		var sb strings.Builder

		if m.cfg.ShowLineNums {
			num := fmt.Sprintf("%*d", digitCount, row+1)
			numStyle := m.cfg.Style.LineNum
			if m.focused && row == cursor.Row {
				numStyle = m.cfg.Style.LineNumActive
			}
			sb.WriteString(numStyle.Render(num))
			sb.WriteString(m.cfg.Style.Gutter.Render(" "))
		}

		vt := m.virtualTextForRow(row, line)
		vt = m.virtualTextWithGhost(row, line, vt)

		var highlights []HighlightSpan
		if m.cfg.Highlighter != nil && row >= highlightStartRow && row < highlightEndRow {
			visible, rawToVisible := visibleTextAfterDeletions(line, vt)
			visLen := len([]rune(visible))

			hasCursor := cursor.Row == row
			cursorCol := -1
			rawCursorCol := -1
			if hasCursor {
				rawLen := len([]rune(line))
				rawCursorCol = clampInt(cursor.Col, 0, rawLen)
				if rawCursorCol >= 0 && rawCursorCol < len(rawToVisible) {
					cursorCol = clampInt(rawToVisible[rawCursorCol], 0, visLen)
				} else {
					cursorCol = visLen
				}
			}

			spans, err := m.cfg.Highlighter.HighlightLine(LineContext{
				Row:          row,
				RawText:      line,
				Text:         visible,
				CursorCol:    cursorCol,
				RawCursorCol: rawCursorCol,
				HasCursor:    hasCursor,
			})
			if err == nil {
				highlights = normalizeHighlightSpans(spans, visLen)
			}
		}

		vl := BuildVisualLine(line, vt, m.cfg.TabWidth)
		sb.WriteString(renderVisualLine(m.cfg.Style, vl, row, cursor, m.focused, sel, selOK, highlights, xOffset, contentWidth))

		out = append(out, sb.String())
	}

	return strings.Join(out, "\n")
}

func renderVisualLine(st Style, vl VisualLine, row int, cursor buffer.Pos, focused bool, sel buffer.Range, selOK bool, highlights []HighlightSpan, xOffset int, contentWidth int) string {
	rawLen := vl.RawLen

	cursorCol := cursor.Col
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
			if cursorCol >= tok.DocStartCol && cursorCol < tok.DocEndCol {
				cursorTokenIdx = i
				break
			}
		}
		if cursorTokenIdx == -1 {
			// Cursor is inside a deleted range; snap to the next visible doc-backed token.
			targetCell := vl.VisualCellForDocCol(cursorCol)
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

	left := 0
	right := int(^uint(0) >> 1) // MaxInt
	if contentWidth > 0 {
		left = maxInt(xOffset, 0)
		right = left + contentWidth
	}

	isAllSpaces := func(s string) bool {
		if s == "" {
			return false
		}
		for _, r := range s {
			if r != ' ' {
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
		splittable := isAllSpaces(tok.Text) && tok.CellWidth == len([]rune(tok.Text))

		write := func(s string) { sb.WriteString(s) }

		switch tok.Kind {
		case VisualTokenVirtual:
			style := st.Text
			switch tok.Role {
			case VirtualRoleGhost:
				style = st.Ghost.Inherit(st.Text)
			case VirtualRoleOverlay:
				style = st.VirtualOverlay.Inherit(st.Text)
			}
			write(renderSpan(style.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
		case VisualTokenDoc:
			selected := hasSel && tok.DocStartCol < selEndCol && tok.DocEndCol > selStartCol
			if hasCursor && i == cursorTokenIdx {
				write(renderSpan(st.Cursor.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			} else if selected {
				write(renderSpan(st.Selection.Render, tok.Text, tok.CellWidth, spanStart, spanWidth, splittable))
			} else {
				style := st.Text
				for _, sp := range highlights {
					if tok.VisibleStartCol < sp.EndCol && tok.VisibleEndCol > sp.StartCol {
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
