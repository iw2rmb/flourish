package editor

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/iw2rmb/flourish/buffer"
	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

// LinkSpan is a hyperlink span over one raw document line.
//
// StartGraphemeCol and EndGraphemeCol are grapheme indices in RawText,
// half-open [StartGraphemeCol, EndGraphemeCol).
type LinkSpan struct {
	StartGraphemeCol int
	EndGraphemeCol   int
	Target           string
	Style            *lipgloss.Style
}

// LinkContext is passed to LinkProvider for one logical line.
type LinkContext struct {
	Row int

	// RawText is the buffer line text (unwrapped) before virtual deletions.
	RawText string
	// Text is the visible line text (unwrapped) after applying virtual deletions.
	Text string

	// CursorGraphemeCol is the grapheme index within Text (visible line), if cursor is on this row; otherwise -1.
	CursorGraphemeCol int
	// RawCursorGraphemeCol is the grapheme index within RawText (buffer line), if cursor is on this row; otherwise -1.
	RawCursorGraphemeCol int
	HasCursor            bool

	// Useful for caching.
	DocID      string
	DocVersion uint64
}

// LinkProvider optionally returns hyperlink spans for a visible line.
//
// Spans are line-local and are interpreted in raw grapheme columns.
// Invalid or overlapping ranges are sanitized deterministically.
type LinkProvider func(ctx LinkContext) ([]LinkSpan, error)

// LinkHit is the resolved hyperlink at a document position.
type LinkHit struct {
	Row int

	StartGraphemeCol int
	EndGraphemeCol   int

	Target string
}

type resolvedLinkSpan struct {
	StartRawGraphemeCol int
	EndRawGraphemeCol   int

	StartVisibleGraphemeCol int
	EndVisibleGraphemeCol   int

	Target string
	Style  lipgloss.Style
}

func normalizeLinkSpans(
	spans []LinkSpan,
	rawLineLen int,
	visibleLineLen int,
	rawToVisible []int,
	defaultStyle lipgloss.Style,
) []resolvedLinkSpan {
	if len(spans) == 0 {
		return nil
	}

	rawLineLen = max(rawLineLen, 0)
	visibleLineLen = max(visibleLineLen, 0)

	out := make([]resolvedLinkSpan, 0, len(spans))
	for _, sp := range spans {
		if sp.Target == "" {
			continue
		}
		start := clampInt(sp.StartGraphemeCol, 0, rawLineLen)
		end := clampInt(sp.EndGraphemeCol, 0, rawLineLen)
		if end < start {
			start, end = end, start
		}
		if start == end {
			continue
		}

		visStart := visibleLineLen
		if start >= 0 && start < len(rawToVisible) {
			visStart = clampInt(rawToVisible[start], 0, visibleLineLen)
		}
		visEnd := visibleLineLen
		if end >= 0 && end < len(rawToVisible) {
			visEnd = clampInt(rawToVisible[end], 0, visibleLineLen)
		}
		if visEnd < visStart {
			visStart, visEnd = visEnd, visStart
		}
		if visStart == visEnd {
			continue
		}

		style := defaultStyle
		if sp.Style != nil {
			style = sp.Style.Inherit(defaultStyle)
		}

		out = append(out, resolvedLinkSpan{
			StartRawGraphemeCol: start,
			EndRawGraphemeCol:   end,

			StartVisibleGraphemeCol: visStart,
			EndVisibleGraphemeCol:   visEnd,

			Target: sp.Target,
			Style:  style,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StartRawGraphemeCol != out[j].StartRawGraphemeCol {
			return out[i].StartRawGraphemeCol < out[j].StartRawGraphemeCol
		}
		return out[i].EndRawGraphemeCol < out[j].EndRawGraphemeCol
	})

	// Keep non-overlapping spans deterministically.
	merged := make([]resolvedLinkSpan, 0, len(out))
	for _, sp := range out {
		if len(merged) == 0 {
			merged = append(merged, sp)
			continue
		}
		last := merged[len(merged)-1]
		if sp.StartRawGraphemeCol < last.EndRawGraphemeCol {
			continue
		}
		merged = append(merged, sp)
	}

	return merged
}

func sanitizeHyperlinkTarget(target string) string {
	if target == "" {
		return ""
	}
	target = strings.ReplaceAll(target, "\x1b", "")
	target = strings.ReplaceAll(target, "\a", "")
	target = strings.ReplaceAll(target, "\r", "")
	target = strings.ReplaceAll(target, "\n", "")
	return target
}

func renderHyperlink(target, text string) string {
	target = sanitizeHyperlinkTarget(target)
	if target == "" || text == "" {
		return text
	}
	const st = "\x1b\\"
	return "\x1b]8;;" + target + st + text + "\x1b]8;;" + st
}

func (m *Model) linksForLine(row int, rawLine string, vt VirtualText, cursor buffer.Pos) []resolvedLinkSpan {
	if m.cfg.LinkProvider == nil || m.buf == nil {
		return nil
	}

	visible, rawToVisible := visibleTextAfterDeletions(rawLine, vt)
	rawLen := graphemeutil.Count(rawLine)
	visLen := graphemeutil.Count(visible)

	hasCursor := cursor.Row == row
	cursorCol := -1
	rawCursorCol := -1
	if hasCursor {
		rawCursorCol = clampInt(cursor.GraphemeCol, 0, rawLen)
		if rawCursorCol >= 0 && rawCursorCol < len(rawToVisible) {
			cursorCol = clampInt(rawToVisible[rawCursorCol], 0, visLen)
		} else {
			cursorCol = visLen
		}
	}

	spans, err := m.cfg.LinkProvider(LinkContext{
		Row:                  row,
		RawText:              rawLine,
		Text:                 visible,
		CursorGraphemeCol:    cursorCol,
		RawCursorGraphemeCol: rawCursorCol,
		HasCursor:            hasCursor,
		DocID:                m.cfg.DocID,
		DocVersion:           m.buf.Version(),
	})
	if err != nil {
		return nil
	}

	return normalizeLinkSpans(spans, rawLen, visLen, rawToVisible, m.cfg.Style.Link)
}

func (m *Model) linkAtDocPos(pos buffer.Pos) (LinkHit, bool) {
	if m.buf == nil {
		return LinkHit{}, false
	}
	m.syncFromBuffer()

	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	if pos.Row < 0 || pos.Row >= len(layout.lines) {
		return LinkHit{}, false
	}

	line := layout.lines[pos.Row]
	if !line.linksResolved {
		line.links = m.linksForLine(pos.Row, line.rawLine, line.vt, m.buf.Cursor())
		line.linksResolved = true
		m.layout.lines[pos.Row] = line
	}
	col := clampInt(pos.GraphemeCol, 0, line.visual.RawGraphemeLen)
	for _, link := range line.links {
		if col < link.StartRawGraphemeCol || col >= link.EndRawGraphemeCol {
			continue
		}
		return LinkHit{
			Row:              pos.Row,
			StartGraphemeCol: link.StartRawGraphemeCol,
			EndGraphemeCol:   link.EndRawGraphemeCol,
			Target:           link.Target,
		}, true
	}
	return LinkHit{}, false
}

// LinkAt returns hyperlink metadata for a document position.
func (m Model) LinkAt(pos buffer.Pos) (LinkHit, bool) {
	return (&m).linkAtDocPos(pos)
}

// LinkAtScreen returns hyperlink metadata for viewport-local screen coordinates.
func (m Model) LinkAtScreen(x, y int) (LinkHit, bool) {
	p := (&m).screenToDocPos(x, y)
	return (&m).linkAtDocPos(p)
}
