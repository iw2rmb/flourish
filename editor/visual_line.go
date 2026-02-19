package editor

import (
	"strings"

	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

type VisualTokenKind int

const (
	VisualTokenDoc VisualTokenKind = iota
	VisualTokenVirtual
)

type VisualToken struct {
	Kind VisualTokenKind

	// Text is the rendered token text. Tabs are expanded to spaces.
	Text string

	// StartCell is the visual cell offset where this token begins.
	StartCell int

	// CellWidth is the number of terminal cells this token occupies.
	CellWidth int

	// VisibleStartGraphemeCol/VisibleEndGraphemeCol define the grapheme span in the visible line text
	// (after deletions) that this token corresponds to. Meaningful only for doc tokens.
	VisibleStartGraphemeCol int
	VisibleEndGraphemeCol   int

	// DocStartGraphemeCol/DocEndGraphemeCol define the raw document grapheme span this token corresponds to.
	// For virtual tokens, DocStartGraphemeCol == DocEndGraphemeCol == AnchorCol.
	DocStartGraphemeCol int
	DocEndGraphemeCol   int

	// Role is meaningful only for virtual tokens.
	Role VirtualRole
	// StyleKey is meaningful only for virtual tokens.
	StyleKey string
}

type VisualLine struct {
	RawGraphemeLen int // grapheme length of the raw buffer line

	Tokens []VisualToken

	// VisualCellToDocGraphemeCol maps each visual cell to a raw document grapheme column.
	// For wide graphemes, every cell maps to the same doc column.
	// For virtual insertions, every cell maps to the insertion anchor column.
	VisualCellToDocGraphemeCol []int

	// DocGraphemeColToVisualCell maps raw document grapheme columns to a visual cell offset.
	// Deleted doc columns map to the next visible doc column (or EOL if none).
	DocGraphemeColToVisualCell []int
}

func BuildVisualLine(rawLine string, vt VirtualText, tabWidth int) VisualLine {
	rawGraphemes := graphemeutil.Split(rawLine)
	rawLen := len(rawGraphemes)
	if tabWidth <= 0 {
		tabWidth = 4
	}

	vt = normalizeVirtualText(vt, rawLen)

	deleted := make([]bool, rawLen)
	for _, d := range vt.Deletions {
		for i := d.StartGraphemeCol; i < d.EndGraphemeCol && i < rawLen; i++ {
			if i >= 0 {
				deleted[i] = true
			}
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

	type docGrapheme struct {
		rawStart int
		rawEnd   int
		visStart int
		visEnd   int
		text     string
		width    int // computed later (tabs depend on visual col)
	}

	docGraphemes := make([]docGrapheme, 0, len(visibleGraphemes))
	for visIdx, gr := range visibleGraphemes {
		rawStart := visibleRawCols[visIdx]
		docGraphemes = append(docGraphemes, docGrapheme{
			rawStart: rawStart,
			rawEnd:   rawStart + 1,
			visStart: visIdx,
			visEnd:   visIdx + 1,
			text:     gr,
		})
	}

	ins := vt.Insertions
	insIdx := 0

	var tokens []VisualToken
	var visualCellToDoc []int
	visualCol := 0

	appendToken := func(kind VisualTokenKind, text string, cellWidth int, visibleStart, visibleEnd int, docStart, docEnd int, role VirtualRole, styleKey string) {
		if cellWidth < 1 {
			cellWidth = 1
		}
		if text == "" {
			// Ensure something is rendered for a 1-cell token.
			text = " "
		}
		startCell := len(visualCellToDoc)
		mapCol := docStart
		if kind == VisualTokenDoc {
			mapCol = docStart
		}
		for i := 0; i < cellWidth; i++ {
			visualCellToDoc = append(visualCellToDoc, mapCol)
		}
		tokens = append(tokens, VisualToken{
			Kind:                    kind,
			Text:                    text,
			StartCell:               startCell,
			CellWidth:               cellWidth,
			VisibleStartGraphemeCol: visibleStart,
			VisibleEndGraphemeCol:   visibleEnd,
			DocStartGraphemeCol:     docStart,
			DocEndGraphemeCol:       docEnd,
			Role:                    role,
			StyleKey:                styleKey,
		})
		visualCol += cellWidth
	}

	appendInsertion := func(in VirtualInsertion) {
		for _, gr := range splitGraphemeBoundaries(in.Text) {
			text := gr.Text
			width := graphemeCellWidth(text, visualCol, tabWidth)
			if width < 1 {
				width = 1
			}
			appendToken(VisualTokenVirtual, text, width, -1, -1, in.GraphemeCol, in.GraphemeCol, in.Role, in.StyleKey)
		}
	}

	for _, dg := range docGraphemes {
		// Emit insertions that anchor before (or inside) this doc grapheme.
		for insIdx < len(ins) && ins[insIdx].GraphemeCol < dg.rawEnd {
			appendInsertion(ins[insIdx])
			insIdx++
		}

		text := dg.text
		if text == "\t" {
			adv := tabAdvance(visualCol, tabWidth)
			appendToken(VisualTokenDoc, strings.Repeat(" ", adv), adv, dg.visStart, dg.visEnd, dg.rawStart, dg.rawEnd, 0, "")
			continue
		}
		width := graphemeCellWidth(text, visualCol, tabWidth)
		if width < 1 {
			width = 1
		}
		appendToken(VisualTokenDoc, text, width, dg.visStart, dg.visEnd, dg.rawStart, dg.rawEnd, 0, "")
	}

	// Emit any remaining insertions (including EOL insertions).
	for insIdx < len(ins) {
		appendInsertion(ins[insIdx])
		insIdx++
	}

	visualLen := len(visualCellToDoc)
	docToVisual := make([]int, rawLen+1)
	for i := range docToVisual {
		docToVisual[i] = visualLen
	}
	docToVisual[rawLen] = visualLen

	for _, tok := range tokens {
		if tok.Kind != VisualTokenDoc {
			continue
		}
		start := clampInt(tok.DocStartGraphemeCol, 0, rawLen)
		end := clampInt(tok.DocEndGraphemeCol, 0, rawLen)
		for c := start; c < end; c++ {
			docToVisual[c] = tok.StartCell
		}
	}
	for c := rawLen - 1; c >= 0; c-- {
		if docToVisual[c] == visualLen {
			docToVisual[c] = docToVisual[c+1]
		}
	}

	return VisualLine{
		RawGraphemeLen:             rawLen,
		Tokens:                     tokens,
		VisualCellToDocGraphemeCol: visualCellToDoc,
		DocGraphemeColToVisualCell: docToVisual,
	}
}

func (vl VisualLine) VisualLen() int { return len(vl.VisualCellToDocGraphemeCol) }

func (vl VisualLine) DocGraphemeColForVisualCell(x int) int {
	if len(vl.VisualCellToDocGraphemeCol) == 0 {
		return vl.RawGraphemeLen
	}
	if x < 0 {
		x = 0
	}
	if x >= len(vl.VisualCellToDocGraphemeCol) {
		return vl.RawGraphemeLen
	}
	col := vl.VisualCellToDocGraphemeCol[x]
	return clampInt(col, 0, vl.RawGraphemeLen)
}

func (vl VisualLine) VisualCellForDocGraphemeCol(col int) int {
	if col < 0 {
		col = 0
	}
	if col > vl.RawGraphemeLen {
		col = vl.RawGraphemeLen
	}
	if len(vl.DocGraphemeColToVisualCell) == 0 {
		return 0
	}
	return clampInt(vl.DocGraphemeColToVisualCell[col], 0, len(vl.VisualCellToDocGraphemeCol))
}

func tabAdvance(visualCol, tabWidth int) int {
	if tabWidth <= 0 {
		tabWidth = 4
	}
	mod := visualCol % tabWidth
	adv := tabWidth - mod
	if adv < 1 {
		return 1
	}
	return adv
}
