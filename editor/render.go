package editor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iw2rmb/flouris/buffer"
)

func (m Model) renderContent() string {
	if m.buf == nil {
		return ""
	}

	lines := strings.Split(m.buf.Text(), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	cursor := m.buf.Cursor()
	sel, selOK := m.buf.Selection()
	gutterDigits := 0
	if m.cfg.ShowLineNums {
		gutterDigits = len(strconv.Itoa(len(lines)))
	}

	out := make([]string, 0, len(lines))
	for row, line := range lines {
		var sb strings.Builder

		if m.cfg.ShowLineNums {
			num := fmt.Sprintf("%*d", gutterDigits, row+1)
			numStyle := m.cfg.Style.LineNum
			if m.focused && row == cursor.Row {
				numStyle = m.cfg.Style.LineNumActive
			}
			sb.WriteString(numStyle.Render(num))
			sb.WriteString(m.cfg.Style.Gutter.Render(" "))
		}

		sb.WriteString(renderLine(m.cfg.Style, line, row, cursor, m.focused, sel, selOK))

		out = append(out, sb.String())
	}

	return strings.Join(out, "\n")
}

func renderLine(st Style, line string, row int, cursor buffer.Pos, focused bool, sel buffer.Range, selOK bool) string {
	runes := []rune(line)
	cursorCol := cursor.Col
	if row != cursor.Row || !focused {
		cursorCol = -1
	} else {
		if cursorCol < 0 {
			cursorCol = 0
		}
		if cursorCol > len(runes) {
			cursorCol = len(runes)
		}
	}

	selStartCol, selEndCol := 0, 0
	if selOK && row >= sel.Start.Row && row <= sel.End.Row {
		selStartCol = 0
		selEndCol = len(runes)
		if row == sel.Start.Row {
			selStartCol = sel.Start.Col
		}
		if row == sel.End.Row {
			selEndCol = sel.End.Col
		}
		if selStartCol < 0 {
			selStartCol = 0
		}
		if selEndCol < 0 {
			selEndCol = 0
		}
		if selStartCol > len(runes) {
			selStartCol = len(runes)
		}
		if selEndCol > len(runes) {
			selEndCol = len(runes)
		}
		if selStartCol > selEndCol {
			selStartCol, selEndCol = selEndCol, selStartCol
		}
		if selStartCol == selEndCol {
			selOK = false
		}
	}

	var sb strings.Builder
	i := 0
	for i < len(runes) {
		if cursorCol == i {
			sb.WriteString(st.Cursor.Render(string(runes[i])))
			i++
			continue
		}

		selected := selOK && i >= selStartCol && i < selEndCol
		j := i + 1
		for j < len(runes) && j != cursorCol {
			nextSelected := selOK && j >= selStartCol && j < selEndCol
			if nextSelected != selected {
				break
			}
			j++
		}

		chunk := string(runes[i:j])
		if selected {
			sb.WriteString(st.Selection.Render(chunk))
		} else {
			sb.WriteString(st.Text.Render(chunk))
		}
		i = j
	}
	if cursorCol == len(runes) {
		sb.WriteString(st.Cursor.Render(" "))
	}
	return sb.String()
}
