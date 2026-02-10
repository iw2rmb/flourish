package editor

import (
	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

type graphemeBoundary struct {
	StartCol int
	EndCol   int
	Text     string
}

type graphemeStep struct {
	Col       int
	NextCol   int
	CellWidth int
}

func splitGraphemeBoundaries(text string) []graphemeBoundary {
	if text == "" {
		return nil
	}

	out := make([]graphemeBoundary, 0, len([]rune(text)))
	g := uniseg.NewGraphemes(text)
	col := 0
	for g.Next() {
		r := g.Runes()
		if len(r) == 0 {
			continue
		}
		out = append(out, graphemeBoundary{
			StartCol: col,
			EndCol:   col + len(r),
			Text:     g.Str(),
		})
		col += len(r)
	}
	return out
}

// iterateGraphemeSteps provides (doc rune index) -> (next rune index, cell width)
// for a single logical line segment. Widths are terminal-cell widths.
func iterateGraphemeSteps(text string, tabWidth int, startCell int) []graphemeStep {
	bounds := splitGraphemeBoundaries(text)
	if len(bounds) == 0 {
		return nil
	}

	out := make([]graphemeStep, 0, len(bounds))
	visualCol := maxInt(startCell, 0)
	for _, b := range bounds {
		w := graphemeCellWidth(b.Text, visualCol, tabWidth)
		if w < 0 {
			w = 0
		}
		out = append(out, graphemeStep{
			Col:       b.StartCol,
			NextCol:   b.EndCol,
			CellWidth: w,
		})
		visualCol += w
	}
	return out
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
