package editor

import (
	"strconv"
	"strings"

	"github.com/iw2rmb/flouris/buffer"
)

func gutterDigits(lineCount int) int {
	if lineCount < 1 {
		lineCount = 1
	}
	return len(strconv.Itoa(lineCount))
}

func (m Model) gutterWidth(lineCount int) int {
	if !m.cfg.ShowLineNums {
		return 0
	}
	// Rendered as: "%*d" + " " (gutter spacer).
	return gutterDigits(lineCount) + 1
}

// screenToDocPos maps viewport-local mouse coordinates to a document position.
//
// Coordinates are in terminal cells and are relative to the editor's viewport:
// (0,0) is the top-left of the visible content region.
//
// v0 mapping rules:
// - runes are treated as 1 cell wide
// - gutter clicks map to col 0
// - x/y are clamped into document bounds
func (m Model) screenToDocPos(x, y int) buffer.Pos {
	if m.buf == nil {
		return buffer.Pos{}
	}

	lines := strings.Split(m.buf.Text(), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	row := m.viewport.YOffset + y
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		row = len(lines) - 1
	}

	if x < 0 {
		x = 0
	}
	col := x - m.gutterWidth(len(lines))
	if col < 0 {
		col = 0
	}

	runes := []rune(lines[row])
	if col > len(runes) {
		col = len(runes)
	}

	return buffer.Pos{Row: row, Col: col}
}

