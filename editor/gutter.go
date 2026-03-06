package editor

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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

// RowMarkProvider resolves inserted/updated/deleted markers for one rendered row.
type RowMarkProvider func(ctx RowMarkContext) RowMarkState

// RowMarkContext describes a rendered visual row for marker resolution.
type RowMarkContext struct {
	Row          int
	SegmentIndex int
	// LineText is raw document line text (unwrapped, before virtual transforms).
	LineText    string
	IsCursorRow bool
	Focused     bool
	DocID       string
	DocVersion  uint64
}

// RowMarkState describes change markers to render for a row.
//
// Deleted markers are rendered only on segment index 0.
// Inserted/Updated markers are rendered on all visual segments of a wrapped row.
type RowMarkState struct {
	Inserted     bool
	Updated      bool
	DeletedAbove bool
	DeletedBelow bool
}

// RowMarkSymbols configures default marker glyphs.
type RowMarkSymbols struct {
	Inserted     string // default: "▎"
	Updated      string // default: "▎"
	DeletedAbove string // default: "▼"
	DeletedBelow string // default: "▲"
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
			Text:     spaceString(ctx.Width),
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

func defaultRowMarkSymbols() RowMarkSymbols {
	return RowMarkSymbols{
		Inserted:     "▎",
		Updated:      "▎",
		DeletedAbove: "▼",
		DeletedBelow: "▲",
	}
}

func normalizeRowMarkSymbols(s RowMarkSymbols) RowMarkSymbols {
	def := defaultRowMarkSymbols()
	if s.Inserted == "" {
		s.Inserted = def.Inserted
	}
	if s.Updated == "" {
		s.Updated = def.Updated
	}
	if s.DeletedAbove == "" {
		s.DeletedAbove = def.DeletedAbove
	}
	if s.DeletedBelow == "" {
		s.DeletedBelow = def.DeletedBelow
	}
	return s
}

func (m Model) resolvedBaseGutterWidth(lineCount int) int {
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

func (m Model) resolvedRowMarkWidth() int {
	if m.cfg.RowMarkProvider == nil {
		return 0
	}
	if m.cfg.RowMarkWidth <= 0 {
		return 2
	}
	return m.cfg.RowMarkWidth
}

func (m Model) resolvedGutterWidth(lineCount int) int {
	return m.resolvedBaseGutterWidth(lineCount) + m.resolvedRowMarkWidth()
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

func (m Model) resolveRowMarkCell(row, segmentIndex int, lineText string, width int, isCursorRow bool) GutterCell {
	if width <= 0 {
		return GutterCell{}
	}

	state := RowMarkState{}
	if m.cfg.RowMarkProvider != nil {
		state = m.cfg.RowMarkProvider(RowMarkContext{
			Row:          row,
			SegmentIndex: segmentIndex,
			LineText:     lineText,
			IsCursorRow:  isCursorRow,
			Focused:      m.focused,
			DocID:        m.cfg.DocID,
			DocVersion:   m.docVersion(),
		})
	}

	symbol, style, ok := m.rowMarkVisualForState(state, segmentIndex)
	if !ok {
		return GutterCell{
			Segments: normalizeGutterSegments(nil, width),
		}
	}

	seg := GutterSegment{Text: symbol}
	segStyle := style
	seg.Style = &segStyle
	return GutterCell{
		Segments: normalizeGutterSegments([]GutterSegment{seg}, width),
	}
}

func (m Model) rowMarkVisualForState(state RowMarkState, segmentIndex int) (symbol string, style lipgloss.Style, ok bool) {
	syms := m.cfg.RowMarkSymbols

	if state.DeletedAbove && segmentIndex == 0 {
		return syms.DeletedAbove, m.cfg.Style.RowMarkDeleted, true
	}
	if state.DeletedBelow && segmentIndex == 0 {
		return syms.DeletedBelow, m.cfg.Style.RowMarkDeleted, true
	}
	if state.Inserted {
		return syms.Inserted, m.cfg.Style.RowMarkInserted, true
	}
	if state.Updated {
		return syms.Updated, m.cfg.Style.RowMarkUpdated, true
	}
	return "", lipgloss.Style{}, false
}

func normalizeGutterSegments(in []GutterSegment, width int) []GutterSegment {
	if width <= 0 {
		return nil
	}

	used := 0
	out := make([]GutterSegment, 0, len(in)+1)
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
		text := sanitizeSegmentText(seg.Text)
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
				appendSegment(seg, spaceString(remaining))
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
		out = append(out, GutterSegment{Text: spaceString(width - used)})
	}
	if len(out) == 0 {
		out = append(out, GutterSegment{Text: spaceString(width)})
	}
	return out
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
