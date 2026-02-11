package editor

import (
	graphemeutil "github.com/iw2rmb/flouris/internal/grapheme"
)

type wrappedSegment struct {
	StartGraphemeCol int
	EndGraphemeCol   int
	Cells            int

	startCell int
	endCell   int
}

type wrapUnit struct {
	startCell int
	endCell   int
	width     int

	isWhitespace bool
	isPunct      bool
}

func wrapSegmentsForVisualLine(vl VisualLine, mode WrapMode, width int) []wrappedSegment {
	visualLen := vl.VisualLen()
	if width <= 0 || mode == WrapNone {
		return []wrappedSegment{{
			StartGraphemeCol: 0,
			EndGraphemeCol:   vl.RawGraphemeLen,
			Cells:            visualLen,
			startCell:        0,
			endCell:          visualLen,
		}}
	}

	units := wrapUnitsFromVisualLine(vl)
	if len(units) == 0 {
		return []wrappedSegment{{
			StartGraphemeCol: 0,
			EndGraphemeCol:   0,
			Cells:            0,
			startCell:        0,
			endCell:          0,
		}}
	}

	segments := make([]wrappedSegment, 0, 1+visualLen/maxInt(width, 1))
	for start := 0; start < len(units); {
		used := 0
		overflow := start
		for overflow < len(units) {
			w := maxInt(units[overflow].width, 1)
			if used > 0 && used+w > width {
				break
			}
			used += w
			overflow++
		}

		if overflow <= start {
			overflow = minInt(start+1, len(units))
		}

		end := overflow
		if mode == WrapWord && overflow < len(units) {
			if br, ok := findWordWrapBreak(units, start, overflow); ok {
				end = br
			} else {
				end = adjustBreakForLeadingPunctuation(units, start, overflow)
			}
		}
		if end <= start {
			end = minInt(start+1, len(units))
		}

		seg := segmentFromUnitRange(vl, units, start, end)
		segments = append(segments, seg)
		start = end
	}

	return segments
}

func wrapUnitsFromVisualLine(vl VisualLine) []wrapUnit {
	if len(vl.Tokens) == 0 {
		return nil
	}

	units := make([]wrapUnit, 0, len(vl.Tokens))
	for _, tok := range vl.Tokens {
		if tok.CellWidth <= 0 {
			continue
		}

		splittableSpaces := tokenIsSplittableSpaces(tok)
		if splittableSpaces {
			for c := 0; c < tok.CellWidth; c++ {
				startCell := tok.StartCell + c
				units = append(units, wrapUnit{
					startCell:    startCell,
					endCell:      startCell + 1,
					width:        1,
					isWhitespace: true,
					isPunct:      false,
				})
			}
			continue
		}

		isWhitespace, isPunct := tokenClass(tok.Text)
		units = append(units, wrapUnit{
			startCell:    tok.StartCell,
			endCell:      tok.StartCell + tok.CellWidth,
			width:        tok.CellWidth,
			isWhitespace: isWhitespace,
			isPunct:      isPunct,
		})
	}

	return units
}

func tokenIsSplittableSpaces(tok VisualToken) bool {
	if tok.Text == "" {
		return false
	}
	if tok.CellWidth != graphemeutil.Count(tok.Text) {
		return false
	}
	for _, gr := range graphemeutil.Split(tok.Text) {
		if !graphemeutil.IsSpace(gr) {
			return false
		}
	}
	return true
}

func tokenClass(text string) (isWhitespace bool, isPunct bool) {
	if text == "" {
		return false, false
	}
	isWhitespace = true
	isPunct = true
	for _, gr := range graphemeutil.Split(text) {
		if !graphemeutil.IsSpace(gr) {
			isWhitespace = false
		}
		if !graphemeutil.IsPunct(gr) {
			isPunct = false
		}
	}
	if isWhitespace {
		isPunct = false
	}
	return isWhitespace, isPunct
}

func segmentFromUnitRange(vl VisualLine, units []wrapUnit, start, end int) wrappedSegment {
	first := units[start]
	last := units[end-1]

	startCell := first.startCell
	endCell := last.endCell
	if endCell < startCell {
		endCell = startCell
	}

	startCol := clampInt(vl.DocGraphemeColForVisualCell(startCell), 0, vl.RawGraphemeLen)
	endCol := startCol
	if endCell >= vl.VisualLen() {
		endCol = vl.RawGraphemeLen
	} else {
		endCol = clampInt(vl.DocGraphemeColForVisualCell(endCell), 0, vl.RawGraphemeLen)
	}
	if endCol < startCol {
		endCol = startCol
	}

	return wrappedSegment{
		StartGraphemeCol: startCol,
		EndGraphemeCol:   endCol,
		Cells:            endCell - startCell,
		startCell:        startCell,
		endCell:          endCell,
	}
}
