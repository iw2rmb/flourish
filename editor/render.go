package editor

import (
	"fmt"
	"strconv"
	"strings"
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

		if m.focused && row == cursor.Row {
			sb.WriteString(renderCursorLine(m.cfg.Style, line, cursor.Col))
		} else {
			sb.WriteString(m.cfg.Style.Text.Render(line))
		}

		out = append(out, sb.String())
	}

	return strings.Join(out, "\n")
}

func renderCursorLine(st Style, line string, cursorCol int) string {
	runes := []rune(line)
	if cursorCol < 0 {
		cursorCol = 0
	}
	if cursorCol > len(runes) {
		cursorCol = len(runes)
	}

	var sb strings.Builder
	sb.WriteString(st.Text.Render(string(runes[:cursorCol])))
	if cursorCol == len(runes) {
		sb.WriteString(st.Cursor.Render(" "))
		return sb.String()
	}

	sb.WriteString(st.Cursor.Render(string(runes[cursorCol])))
	sb.WriteString(st.Text.Render(string(runes[cursorCol+1:])))
	return sb.String()
}
