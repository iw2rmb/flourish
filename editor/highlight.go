package editor

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

type HighlightSpan struct {
	// StartGraphemeCol and EndGraphemeCol are grapheme indices in the visible line text (after deletions),
	// half-open [StartGraphemeCol, EndGraphemeCol).
	StartGraphemeCol int
	EndGraphemeCol   int
	Style            lipgloss.Style
}

type LineContext struct {
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
}

type Highlighter interface {
	HighlightLine(ctx LineContext) ([]HighlightSpan, error)
}

// visibleLineInfo holds the precomputed visible text and raw→visible column
// mapping for a single logical line. It is computed once and shared by both
// highlightForLine and linksForLine to avoid redundant grapheme splitting.
type visibleLineInfo struct {
	visible      string
	rawToVisible []int
	rawLen       int
	visLen       int
}

func computeVisibleLineInfo(rawLine string, vt VirtualText) visibleLineInfo {
	visible, rawToVisible, rawLen, visLen := visibleTextAfterDeletions(rawLine, vt)
	return visibleLineInfo{
		visible:      visible,
		rawToVisible: rawToVisible,
		rawLen:       rawLen,
		visLen:       visLen,
	}
}

func visibleTextAfterDeletions(rawLine string, vt VirtualText) (visible string, rawToVisible []int, rawLen int, visLen int) {
	rawGraphemes := graphemeutil.Split(rawLine)
	rawLen = len(rawGraphemes)
	deleted := make([]bool, rawLen)
	for _, d := range vt.Deletions {
		start := clampInt(d.StartGraphemeCol, 0, rawLen)
		end := clampInt(d.EndGraphemeCol, 0, rawLen)
		if end < start {
			start, end = end, start
		}
		for i := start; i < end && i < rawLen; i++ {
			deleted[i] = true
		}
	}

	visibleGraphemes := make([]string, 0, rawLen)
	visibleRawCols := make([]int, 0, rawLen)
	for i, gr := range rawGraphemes {
		if deleted[i] {
			continue
		}
		visibleGraphemes = append(visibleGraphemes, gr)
		visibleRawCols = append(visibleRawCols, i)
	}

	visLen = len(visibleGraphemes)
	rawToVisible = make([]int, rawLen+1)
	for i := range rawToVisible {
		rawToVisible[i] = visLen
	}
	rawToVisible[rawLen] = visLen
	for visIdx, rawCol := range visibleRawCols {
		if rawCol >= 0 && rawCol <= rawLen {
			rawToVisible[rawCol] = visIdx
		}
	}
	for c := rawLen - 1; c >= 0; c-- {
		if rawToVisible[c] == visLen {
			rawToVisible[c] = rawToVisible[c+1]
		}
	}

	return graphemeutil.Join(visibleGraphemes), rawToVisible, rawLen, visLen
}

func normalizeHighlightSpans(spans []HighlightSpan, lineLen int) []HighlightSpan {
	if len(spans) == 0 {
		return nil
	}
	lineLen = max(lineLen, 0)

	// Fast path: check if spans are already clamped, sorted, non-overlapping, and non-empty.
	normalized := true
	for i, sp := range spans {
		if sp.StartGraphemeCol < 0 || sp.EndGraphemeCol > lineLen || sp.StartGraphemeCol >= sp.EndGraphemeCol {
			normalized = false
			break
		}
		if i > 0 {
			prev := spans[i-1]
			if sp.StartGraphemeCol < prev.EndGraphemeCol ||
				(sp.StartGraphemeCol == prev.StartGraphemeCol && sp.EndGraphemeCol < prev.EndGraphemeCol) {
				normalized = false
				break
			}
		}
	}
	if normalized {
		return spans
	}

	out := make([]HighlightSpan, 0, len(spans))
	for _, sp := range spans {
		start := clampInt(sp.StartGraphemeCol, 0, lineLen)
		end := clampInt(sp.EndGraphemeCol, 0, lineLen)
		if end < start {
			start, end = end, start
		}
		if start == end {
			continue
		}
		out = append(out, HighlightSpan{StartGraphemeCol: start, EndGraphemeCol: end, Style: sp.Style})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StartGraphemeCol != out[j].StartGraphemeCol {
			return out[i].StartGraphemeCol < out[j].StartGraphemeCol
		}
		return out[i].EndGraphemeCol < out[j].EndGraphemeCol
	})

	// v0: enforce non-overlap deterministically by dropping any overlapping spans.
	merged := make([]HighlightSpan, 0, len(out))
	for _, sp := range out {
		if len(merged) == 0 {
			merged = append(merged, sp)
			continue
		}
		last := merged[len(merged)-1]
		if sp.StartGraphemeCol < last.EndGraphemeCol {
			continue
		}
		merged = append(merged, sp)
	}

	return merged
}
