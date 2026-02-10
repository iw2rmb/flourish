package editor

import (
	"fmt"
	"strings"

	"github.com/iw2rmb/flouris/buffer"
)

func (m Model) renderContent() string {
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

	out := make([]string, 0, len(lines))
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
		vl := BuildVisualLine(line, vt, m.cfg.TabWidth)
		sb.WriteString(renderVisualLine(m.cfg.Style, vl, row, cursor, m.focused, sel, selOK))

		out = append(out, sb.String())
	}

	return strings.Join(out, "\n")
}

func renderVisualLine(st Style, vl VisualLine, row int, cursor buffer.Pos, focused bool, sel buffer.Range, selOK bool) string {
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
	eolInsIdx := len(vl.Tokens)
	if renderEOLCursor {
		for i, tok := range vl.Tokens {
			if tok.Kind == VisualTokenVirtual && tok.DocStartCol == rawLen {
				eolInsIdx = i
				break
			}
		}
	}

	var sb strings.Builder
	for i, tok := range vl.Tokens {
		if renderEOLCursor && i == eolInsIdx {
			sb.WriteString(st.Cursor.Render(" "))
		}

		switch tok.Kind {
		case VisualTokenVirtual:
			sb.WriteString(st.Text.Render(tok.Text))
		case VisualTokenDoc:
			selected := hasSel && tok.DocStartCol < selEndCol && tok.DocEndCol > selStartCol
			if hasCursor && i == cursorTokenIdx {
				sb.WriteString(st.Cursor.Render(tok.Text))
			} else if selected {
				sb.WriteString(st.Selection.Render(tok.Text))
			} else {
				sb.WriteString(st.Text.Render(tok.Text))
			}
		default:
			sb.WriteString(st.Text.Render(tok.Text))
		}
	}
	if renderEOLCursor && eolInsIdx == len(vl.Tokens) {
		sb.WriteString(st.Cursor.Render(" "))
	}
	return sb.String()
}
