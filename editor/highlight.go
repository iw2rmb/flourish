package editor

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
)

type HighlightSpan struct {
	// StartCol and EndCol are rune indices in the visible line text (after deletions),
	// half-open [StartCol, EndCol).
	StartCol int
	EndCol   int
	Style    lipgloss.Style
}

type LineContext struct {
	Row int

	// RawText is the buffer line text (unwrapped) before virtual deletions.
	RawText string
	// Text is the visible line text (unwrapped) after applying virtual deletions.
	Text string

	// CursorCol is the rune index within Text (visible line), if cursor is on this row; otherwise -1.
	CursorCol int
	// RawCursorCol is the rune index within RawText (buffer line), if cursor is on this row; otherwise -1.
	RawCursorCol int
	HasCursor    bool
}

type Highlighter interface {
	HighlightLine(ctx LineContext) ([]HighlightSpan, error)
}

func visibleTextAfterDeletions(rawLine string, vt VirtualText) (visible string, rawToVisible []int) {
	rawRunes := []rune(rawLine)
	rawLen := len(rawRunes)
	deleted := make([]bool, rawLen)
	for _, d := range vt.Deletions {
		start := clampInt(d.StartCol, 0, rawLen)
		end := clampInt(d.EndCol, 0, rawLen)
		if end < start {
			start, end = end, start
		}
		for i := start; i < end && i < rawLen; i++ {
			deleted[i] = true
		}
	}

	visibleRunes := make([]rune, 0, rawLen)
	visibleRawCols := make([]int, 0, rawLen)
	for i, r := range rawRunes {
		if deleted[i] {
			continue
		}
		visibleRunes = append(visibleRunes, r)
		visibleRawCols = append(visibleRawCols, i)
	}

	visLen := len(visibleRunes)
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

	return string(visibleRunes), rawToVisible
}

func normalizeHighlightSpans(spans []HighlightSpan, lineLen int) []HighlightSpan {
	if len(spans) == 0 {
		return nil
	}
	lineLen = maxInt(lineLen, 0)

	out := make([]HighlightSpan, 0, len(spans))
	for _, sp := range spans {
		start := clampInt(sp.StartCol, 0, lineLen)
		end := clampInt(sp.EndCol, 0, lineLen)
		if end < start {
			start, end = end, start
		}
		if start == end {
			continue
		}
		out = append(out, HighlightSpan{StartCol: start, EndCol: end, Style: sp.Style})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StartCol != out[j].StartCol {
			return out[i].StartCol < out[j].StartCol
		}
		return out[i].EndCol < out[j].EndCol
	})

	// v0: enforce non-overlap deterministically by dropping any overlapping spans.
	merged := make([]HighlightSpan, 0, len(out))
	for _, sp := range out {
		if len(merged) == 0 {
			merged = append(merged, sp)
			continue
		}
		last := merged[len(merged)-1]
		if sp.StartCol < last.EndCol {
			continue
		}
		merged = append(merged, sp)
	}

	return merged
}

