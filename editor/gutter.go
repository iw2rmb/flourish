package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	// Segments contains style-addressable text chunks for this gutter cell.
	// Segment text is normalized/clipped/padded to the resolved gutter width.
	Segments []GutterSegment
	// ClickCol maps gutter clicks to a document grapheme column.
	// Negative values are clamped to 0.
	ClickCol int
}

type GutterSegment struct {
	Text string
	// StyleKey optionally selects a keyed style via Config.GutterStyleForKey.
	// Empty means use Style.Gutter.
	StyleKey string
	// Style optionally overrides StyleKey for this segment.
	// Returned styles should avoid layout-affecting options (padding/margin/width)
	// to keep render mapping deterministic.
	Style *lipgloss.Style
}

// LineNumberGutter returns the built-in line-number gutter behavior.
func LineNumberGutter() Gutter {
	return Gutter{
		Width: func(ctx GutterWidthContext) int {
			return LineNumberWidth(ctx.LineCount)
		},
		Cell: func(ctx GutterCellContext) GutterCell {
			return GutterCell{
				Segments: []GutterSegment{LineNumberSegment(ctx)},
				ClickCol: 0,
			}
		},
	}
}

// LineNumberWidth returns the default line-number gutter width for lineCount.
func LineNumberWidth(lineCount int) int {
	return gutterDigits(lineCount) + 1
}

// LineNumberSegment returns the default line-number segment for one gutter row.
func LineNumberSegment(ctx GutterCellContext) GutterSegment {
	if ctx.Width <= 0 {
		return GutterSegment{}
	}

	digits := ctx.DigitCount
	if digits < 1 {
		digits = gutterDigits(ctx.LineCount)
	}

	if ctx.SegmentIndex > 0 {
		return GutterSegment{
			Text:     strings.Repeat(" ", ctx.Width),
			StyleKey: "line_num",
		}
	}

	styleKey := "line_num"
	if ctx.Focused && ctx.IsCursorRow {
		styleKey = "line_num_active"
	}

	return GutterSegment{
		Text:     fmt.Sprintf("%*d ", digits, ctx.Row+1),
		StyleKey: styleKey,
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
	cell.Segments = normalizeGutterSegments(cell.Segments, width)
	if cell.ClickCol < 0 {
		cell.ClickCol = 0
	}
	return cell
}

func normalizeGutterSegments(in []GutterSegment, width int) []GutterSegment {
	if width <= 0 {
		return nil
	}

	used := 0
	out := make([]GutterSegment, 0, width)
	appendSegment := func(seg GutterSegment, text string) {
		if text == "" {
			return
		}
		out = append(out, GutterSegment{
			Text:     text,
			StyleKey: seg.StyleKey,
			Style:    seg.Style,
		})
	}

	for _, seg := range in {
		text := sanitizeGutterSegmentText(seg.Text)
		if text == "" {
			continue
		}
		for _, gr := range graphemeutil.Split(text) {
			if used >= width {
				break
			}
			w := graphemeCellWidth(gr, used, 4)
			if w < 1 {
				w = 1
			}
			if used+w > width {
				remaining := width - used
				appendSegment(seg, strings.Repeat(" ", remaining))
				used = width
				break
			}
			appendSegment(seg, gr)
			used += w
		}
		if used >= width {
			break
		}
	}

	if used < width {
		out = append(out, GutterSegment{Text: strings.Repeat(" ", width-used)})
	}
	if len(out) == 0 {
		out = append(out, GutterSegment{Text: strings.Repeat(" ", width)})
	}
	return out
}

func sanitizeGutterSegmentText(s string) string {
	s = sanitizeSingleLine(s)
	if s == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

func resolveGutterSegmentStyle(
	base lipgloss.Style,
	styleForKey func(string) (lipgloss.Style, bool),
	seg GutterSegment,
) lipgloss.Style {
	if seg.Style != nil {
		return seg.Style.Inherit(base)
	}
	if styleForKey != nil && seg.StyleKey != "" {
		if keyed, ok := styleForKey(seg.StyleKey); ok {
			return keyed.Inherit(base)
		}
	}
	return base
}

func renderGutterCell(
	base lipgloss.Style,
	styleForKey func(string) (lipgloss.Style, bool),
	cell GutterCell,
) string {
	if len(cell.Segments) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, seg := range cell.Segments {
		if seg.Text == "" {
			continue
		}
		style := resolveGutterSegmentStyle(base, styleForKey, seg)
		sb.WriteString(style.Render(seg.Text))
	}
	return sb.String()
}

func (m Model) docVersion() uint64 {
	if m.buf == nil {
		return 0
	}
	return m.buf.Version()
}
