package buffer

import "github.com/iw2rmb/flourish/internal/grapheme"

type MoveUnit int

const (
	MoveGrapheme MoveUnit = iota
	MoveWord
	MoveLine
	MoveDoc
)

type MoveDir int

const (
	DirLeft MoveDir = iota
	DirRight
	DirUp
	DirDown
	DirHome // line start (or doc start for MoveDoc)
	DirEnd  // line end (or doc end for MoveDoc)
)

type Move struct {
	Unit MoveUnit
	Dir  MoveDir
	// Count repeats the movement this many times. Values <= 0 are treated as 1.
	Count  int
	Extend bool // if true, updates selection anchor/end; if false clears selection
}

func (b *Buffer) Move(m Move) {
	change := b.beginChange(ChangeSourceLocal)

	prevCursor := b.cursor
	prevSel := b.sel

	usePreferred := usesPreferredColumn(m)
	preferredCol := prevCursor.GraphemeCol
	if usePreferred {
		preferredCol = b.preferredColumn(prevCursor.GraphemeCol)
	}

	nextCursor := b.moveCursor(prevCursor, m, preferredCol, usePreferred)
	nextCursor = b.clampPos(nextCursor)

	nextSel := selectionState{}
	if m.Extend {
		anchor := prevCursor
		if prevSel.active && prevSel.anchor != prevSel.end {
			anchor = prevSel.anchor
		}
		if anchor != nextCursor {
			nextSel = selectionState{active: true, anchor: anchor, end: nextCursor}
		}
	}

	if usePreferred {
		b.setPreferredColumn(preferredCol)
	} else {
		b.setPreferredColumn(nextCursor.GraphemeCol)
	}

	if prevCursor == nextCursor && selectionStateEqual(prevSel, nextSel) {
		return
	}

	b.cursor = nextCursor
	b.sel = nextSel
	b.version++
	b.commitChange(change)
}

func selectionStateEqual(a, b selectionState) bool {
	if !a.active && !b.active {
		return true
	}
	return a.active == b.active && a.anchor == b.anchor && a.end == b.end
}

func usesPreferredColumn(m Move) bool {
	if m.Dir != DirUp && m.Dir != DirDown {
		return false
	}
	return m.Unit == MoveLine || m.Unit == MoveGrapheme
}

func (b *Buffer) moveCursor(p Pos, m Move, preferredCol int, usePreferred bool) Pos {
	count := m.Count
	if count <= 0 {
		count = 1
	}

	next := p
	for i := 0; i < count; i++ {
		prev := next
		switch m.Unit {
		case MoveGrapheme:
			next = b.moveGrapheme(next, m.Dir, preferredCol, usePreferred)
		case MoveWord:
			next = b.moveWord(next, m.Dir)
		case MoveLine:
			next = b.moveLine(next, m.Dir, preferredCol, usePreferred)
		case MoveDoc:
			next = b.moveDoc(next, m.Dir)
		default:
			return next
		}
		if next == prev {
			break
		}
	}
	return next
}

func (b *Buffer) moveGrapheme(p Pos, dir MoveDir, preferredCol int, usePreferred bool) Pos {
	row, col := p.Row, p.GraphemeCol
	lastRow := len(b.lines) - 1

	switch dir {
	case DirLeft:
		if row == 0 && col == 0 {
			return p
		}
		if col > 0 {
			return Pos{Row: row, GraphemeCol: col - 1}
		}
		prevRow := row - 1
		return Pos{Row: prevRow, GraphemeCol: len(b.lines[prevRow])}
	case DirRight:
		if row == lastRow && col == len(b.lines[lastRow]) {
			return p
		}
		if col < len(b.lines[row]) {
			return Pos{Row: row, GraphemeCol: col + 1}
		}
		return Pos{Row: row + 1, GraphemeCol: 0}
	case DirUp, DirDown, DirHome, DirEnd:
		return b.moveLine(p, dir, preferredCol, usePreferred)
	default:
		return p
	}
}

func (b *Buffer) moveWord(p Pos, dir MoveDir) Pos {
	row, col := p.Row, p.GraphemeCol
	line := b.lines[row]

	switch dir {
	case DirLeft:
		return Pos{Row: row, GraphemeCol: prevWordBoundary(line, col)}
	case DirRight:
		return Pos{Row: row, GraphemeCol: nextWordBoundary(line, col)}
	case DirHome:
		return Pos{Row: row, GraphemeCol: 0}
	case DirEnd:
		return Pos{Row: row, GraphemeCol: len(line)}
	default:
		return p
	}
}

func (b *Buffer) moveLine(p Pos, dir MoveDir, preferredCol int, usePreferred bool) Pos {
	row, col := p.Row, p.GraphemeCol
	lastRow := len(b.lines) - 1

	switch dir {
	case DirHome:
		return Pos{Row: row, GraphemeCol: 0}
	case DirEnd:
		return Pos{Row: row, GraphemeCol: len(b.lines[row])}
	case DirUp:
		if row == 0 {
			return p
		}
		nr := row - 1
		targetCol := col
		if usePreferred {
			targetCol = preferredCol
		}
		return Pos{Row: nr, GraphemeCol: min(targetCol, len(b.lines[nr]))}
	case DirDown:
		if row == lastRow {
			return p
		}
		nr := row + 1
		targetCol := col
		if usePreferred {
			targetCol = preferredCol
		}
		return Pos{Row: nr, GraphemeCol: min(targetCol, len(b.lines[nr]))}
	default:
		return p
	}
}

func (b *Buffer) moveDoc(p Pos, dir MoveDir) Pos {
	lastRow := len(b.lines) - 1
	lastCol := len(b.lines[lastRow])

	switch dir {
	case DirHome, DirUp:
		return Pos{Row: 0, GraphemeCol: 0}
	case DirEnd, DirDown:
		return Pos{Row: lastRow, GraphemeCol: lastCol}
	default:
		return p
	}
}

// Word boundary rules (v0):
// - skip whitespace, then skip non-whitespace
// - newline is a hard boundary (so this operates on a single logical line)
func prevWordBoundary(line []string, col int) int {
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}
	i := col
	for i > 0 && grapheme.IsSpace(line[i-1]) {
		i--
	}
	for i > 0 && !grapheme.IsSpace(line[i-1]) {
		i--
	}
	return i
}

func nextWordBoundary(line []string, col int) int {
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}
	i := col
	for i < len(line) && grapheme.IsSpace(line[i]) {
		i++
	}
	for i < len(line) && !grapheme.IsSpace(line[i]) {
		i++
	}
	return i
}
