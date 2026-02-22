package editor

import (
	"strings"

	"github.com/mattn/go-runewidth"

	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
	"github.com/rivo/uniseg"
)

type graphemeBoundary struct {
	StartGraphemeCol int
	EndGraphemeCol   int
	Text             string
}

type graphemeStep struct {
	GraphemeCol     int
	NextGraphemeCol int
	CellWidth       int
}

func splitGraphemeBoundaries(text string) []graphemeBoundary {
	if text == "" {
		return nil
	}

	clusters := graphemeutil.Split(text)
	out := make([]graphemeBoundary, 0, len(clusters))
	for i, c := range clusters {
		out = append(out, graphemeBoundary{
			StartGraphemeCol: i,
			EndGraphemeCol:   i + 1,
			Text:             c,
		})
	}
	return out
}

// iterateGraphemeSteps provides (doc grapheme index) -> (next grapheme index, cell width)
// for a single logical line segment. Widths are terminal-cell widths.
func iterateGraphemeSteps(text string, tabWidth int, startCell int) []graphemeStep {
	bounds := splitGraphemeBoundaries(text)
	if len(bounds) == 0 {
		return nil
	}

	out := make([]graphemeStep, 0, len(bounds))
	visualCol := max(startCell, 0)
	for _, b := range bounds {
		w := graphemeCellWidth(b.Text, visualCol, tabWidth)
		if w < 0 {
			w = 0
		}
		out = append(out, graphemeStep{
			GraphemeCol:     b.StartGraphemeCol,
			NextGraphemeCol: b.EndGraphemeCol,
			CellWidth:       w,
		})
		visualCol += w
	}
	return out
}

func sanitizeSegmentText(s string) string {
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

func graphemeCellWidth(text string, visualCol, tabWidth int) int {
	if text == "\t" {
		return tabAdvance(visualCol, tabWidth)
	}

	w := runewidth.StringWidth(text)
	if w < 0 {
		w = 0
	}
	if w == 0 {
		fallback := uniseg.StringWidth(text)
		if fallback > w {
			w = fallback
		}
	}
	return w
}
