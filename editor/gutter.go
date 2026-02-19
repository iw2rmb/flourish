package editor

import (
	"fmt"
	"strings"

	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

// Gutter configures optional custom gutter rendering and hit-testing.
//
// When Width is nil, gutter rendering is disabled.
// When Cell is nil, gutter cells render as blanks and clicks map to col 0.
type Gutter struct {
	Width func(ctx GutterWidthContext) int
	Cell  func(ctx GutterCellContext) GutterCell
}

type GutterWidthContext struct {
	LineCount  int
	WrapMode   WrapMode
	Focused    bool
	DocID      string
	DocVersion uint64
}

type GutterCellContext struct {
	Row          int
	SegmentIndex int
	// LineText is raw document line text (unwrapped, before virtual transforms).
	LineText    string
	Width       int
	DigitCount  int
	LineCount   int
	IsCursorRow bool
	Focused     bool
	DocID       string
	DocVersion  uint64
}

type GutterCell struct {
	Text string
	// StyleKey optionally selects a keyed style via Config.GutterStyleForKey.
	// Empty means use Style.Gutter.
	StyleKey string
	// ClickCol maps gutter clicks to a document grapheme column.
	// Negative values are clamped to 0.
	ClickCol int
}

// LineNumberGutter returns the built-in line-number gutter behavior.
func LineNumberGutter() Gutter {
	return Gutter{
		Width: lineNumberGutterWidth,
		Cell:  lineNumberGutterCell,
	}
}

func lineNumberGutterWidth(ctx GutterWidthContext) int {
	return gutterDigits(ctx.LineCount) + 1
}

func lineNumberGutterCell(ctx GutterCellContext) GutterCell {
	if ctx.Width <= 0 {
		return GutterCell{}
	}

	digits := ctx.DigitCount
	if digits < 1 {
		digits = gutterDigits(ctx.LineCount)
	}

	if ctx.SegmentIndex > 0 {
		return GutterCell{
			Text:     strings.Repeat(" ", ctx.Width),
			StyleKey: "line_num",
			ClickCol: 0,
		}
	}

	styleKey := "line_num"
	if ctx.Focused && ctx.IsCursorRow {
		styleKey = "line_num_active"
	}

	return GutterCell{
		Text:     fmt.Sprintf("%*d ", digits, ctx.Row+1),
		StyleKey: styleKey,
		ClickCol: 0,
	}
}

func gutterDigits(lineCount int) int {
	if lineCount < 1 {
		lineCount = 1
	}
	return len(fmt.Sprintf("%d", lineCount))
}

func (m Model) resolvedGutterWidth(lineCount int) int {
	if m.cfg.Gutter.Width == nil {
		return 0
	}
	if lineCount < 1 {
		lineCount = 1
	}
	w := m.cfg.Gutter.Width(GutterWidthContext{
		LineCount:  lineCount,
		WrapMode:   m.cfg.WrapMode,
		Focused:    m.focused,
		DocID:      m.cfg.DocID,
		DocVersion: m.docVersion(),
	})
	if w < 0 {
		return 0
	}
	return w
}

func (m Model) resolveGutterCell(row, segmentIndex int, lineText string, lineCount, width int, isCursorRow bool) GutterCell {
	if width <= 0 {
		return GutterCell{}
	}
	cell := GutterCell{}
	if m.cfg.Gutter.Cell != nil {
		cell = m.cfg.Gutter.Cell(GutterCellContext{
			Row:          row,
			SegmentIndex: segmentIndex,
			LineText:     lineText,
			Width:        width,
			DigitCount:   gutterDigits(lineCount),
			LineCount:    lineCount,
			IsCursorRow:  isCursorRow,
			Focused:      m.focused,
			DocID:        m.cfg.DocID,
			DocVersion:   m.docVersion(),
		})
	}
	cell.Text = normalizeGutterText(cell.Text, width)
	if cell.ClickCol < 0 {
		cell.ClickCol = 0
	}
	return cell
}

func normalizeGutterText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = sanitizeSingleLine(text)
	if text == "" {
		return strings.Repeat(" ", width)
	}

	var sb strings.Builder
	used := 0
	for _, gr := range graphemeutil.Split(text) {
		w := graphemeCellWidth(gr, used, 4)
		if w < 0 {
			w = 0
		}
		if used+w > width {
			sb.WriteString(strings.Repeat(" ", width-used))
			used = width
			break
		}
		sb.WriteString(gr)
		used += w
		if used == width {
			break
		}
	}
	if used < width {
		sb.WriteString(strings.Repeat(" ", width-used))
	}
	return sb.String()
}

func (m Model) docVersion() uint64 {
	if m.buf == nil {
		return 0
	}
	return m.buf.Version()
}
