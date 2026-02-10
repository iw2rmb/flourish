package editor

import (
	"strings"
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

	// VisibleStartCol/VisibleEndCol define the rune span in the visible line text
	// (after deletions) that this token corresponds to. Meaningful only for doc tokens.
	VisibleStartCol int
	VisibleEndCol   int

	// DocStartCol/DocEndCol define the raw document rune span this token corresponds to.
	// For virtual tokens, DocStartCol == DocEndCol == AnchorCol.
	DocStartCol int
	DocEndCol   int

	// Role is meaningful only for virtual tokens.
	Role VirtualRole
}

type VisualLine struct {
	RawLen int // rune length of the raw buffer line

	Tokens []VisualToken

	// VisualCellToDocCol maps each visual cell to a raw document rune column.
	// For wide graphemes, every cell maps to the same doc column.
	// For virtual insertions, every cell maps to the insertion anchor column.
	VisualCellToDocCol []int

	// DocColToVisualCell maps raw document rune columns to a visual cell offset.
	// Deleted doc columns map to the next visible doc column (or EOL if none).
	DocColToVisualCell []int
}

func BuildVisualLine(rawLine string, vt VirtualText, tabWidth int) VisualLine {
	rawRunes := []rune(rawLine)
	rawLen := len(rawRunes)
	if tabWidth <= 0 {
		tabWidth = 4
	}

	vt = normalizeVirtualText(vt, rawLen)

	deleted := make([]bool, rawLen)
	for _, d := range vt.Deletions {
		for i := d.StartCol; i < d.EndCol && i < rawLen; i++ {
			if i >= 0 {
				deleted[i] = true
			}
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

	visibleText := string(visibleRunes)

	type docGrapheme struct {
		rawStart int
		rawEnd   int
		visStart int
		visEnd   int
		text     string
		width    int // computed later (tabs depend on visual col)
	}

	docGraphemes := make([]docGrapheme, 0, len(visibleRunes))
	if visibleText != "" {
		for _, gr := range splitGraphemeBoundaries(visibleText) {
			rawStart := visibleRawCols[gr.StartCol]
			rawEnd := visibleRawCols[gr.EndCol-1] + 1
			docGraphemes = append(docGraphemes, docGrapheme{
				rawStart: rawStart,
				rawEnd:   rawEnd,
				visStart: gr.StartCol,
				visEnd:   gr.EndCol,
				text:     gr.Text,
			})
		}
	}

	ins := vt.Insertions
	insIdx := 0

	var tokens []VisualToken
	var visualCellToDoc []int
	visualCol := 0

	appendToken := func(kind VisualTokenKind, text string, cellWidth int, visibleStart, visibleEnd int, docStart, docEnd int, role VirtualRole) {
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
			Kind:            kind,
			Text:            text,
			StartCell:       startCell,
			CellWidth:       cellWidth,
			VisibleStartCol: visibleStart,
			VisibleEndCol:   visibleEnd,
			DocStartCol:     docStart,
			DocEndCol:       docEnd,
			Role:            role,
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
			appendToken(VisualTokenVirtual, text, width, -1, -1, in.Col, in.Col, in.Role)
		}
	}

	for _, dg := range docGraphemes {
		// Emit insertions that anchor before (or inside) this doc grapheme.
		for insIdx < len(ins) && ins[insIdx].Col < dg.rawEnd {
			appendInsertion(ins[insIdx])
			insIdx++
		}

		text := dg.text
		if text == "\t" {
			adv := tabAdvance(visualCol, tabWidth)
			appendToken(VisualTokenDoc, strings.Repeat(" ", adv), adv, dg.visStart, dg.visEnd, dg.rawStart, dg.rawEnd, 0)
			continue
		}
		width := graphemeCellWidth(text, visualCol, tabWidth)
		if width < 1 {
			width = 1
		}
		appendToken(VisualTokenDoc, text, width, dg.visStart, dg.visEnd, dg.rawStart, dg.rawEnd, 0)
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
		start := clampInt(tok.DocStartCol, 0, rawLen)
		end := clampInt(tok.DocEndCol, 0, rawLen)
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
		RawLen:             rawLen,
		Tokens:             tokens,
		VisualCellToDocCol: visualCellToDoc,
		DocColToVisualCell: docToVisual,
	}
}

func (vl VisualLine) VisualLen() int { return len(vl.VisualCellToDocCol) }

func (vl VisualLine) DocColForVisualCell(x int) int {
	if len(vl.VisualCellToDocCol) == 0 {
		return vl.RawLen
	}
	if x < 0 {
		x = 0
	}
	if x >= len(vl.VisualCellToDocCol) {
		return vl.RawLen
	}
	col := vl.VisualCellToDocCol[x]
	return clampInt(col, 0, vl.RawLen)
}

func (vl VisualLine) VisualCellForDocCol(col int) int {
	if col < 0 {
		col = 0
	}
	if col > vl.RawLen {
		col = vl.RawLen
	}
	if len(vl.DocColToVisualCell) == 0 {
		return 0
	}
	return clampInt(vl.DocColToVisualCell[col], 0, len(vl.VisualCellToDocCol))
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
