package buffer

import "strings"

func (b *Buffer) InsertText(s string) {
	if s == "" {
		if _, ok := b.Selection(); ok {
			b.DeleteSelection()
		}
		return
	}

	prev := b.snapshot()

	r, ok := b.Selection()
	if !ok {
		r = Range{Start: b.cursor, End: b.cursor}
	}

	nextCursor, changed := b.replaceRange(r, s)
	if !changed {
		return
	}

	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
}

func (b *Buffer) InsertRune(r rune) {
	b.InsertText(string(r))
}

func (b *Buffer) InsertNewline() {
	b.InsertText("\n")
}

func (b *Buffer) DeleteBackward() {
	if _, ok := b.Selection(); ok {
		b.DeleteSelection()
		return
	}

	row, col := b.cursor.Row, b.cursor.Col
	if row == 0 && col == 0 {
		return
	}

	prev := b.snapshot()

	if col > 0 {
		start := Pos{Row: row, Col: col - 1}
		end := Pos{Row: row, Col: col}
		nextCursor, changed := b.replaceRange(Range{Start: start, End: end}, "")
		if !changed {
			return
		}
		b.cursor = nextCursor
		b.sel = selectionState{}
		b.version++
		b.recordUndo(prev)
		return
	}

	// Join with previous line (delete the newline).
	prevRow := row - 1
	start := Pos{Row: prevRow, Col: len(b.lines[prevRow])}
	end := Pos{Row: row, Col: 0}
	nextCursor, changed := b.replaceRange(Range{Start: start, End: end}, "")
	if !changed {
		return
	}
	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
}

func (b *Buffer) DeleteForward() {
	if _, ok := b.Selection(); ok {
		b.DeleteSelection()
		return
	}

	row, col := b.cursor.Row, b.cursor.Col
	lastRow := len(b.lines) - 1
	if row == lastRow && col == len(b.lines[lastRow]) {
		return
	}

	prev := b.snapshot()

	if col < len(b.lines[row]) {
		start := Pos{Row: row, Col: col}
		end := Pos{Row: row, Col: col + 1}
		nextCursor, changed := b.replaceRange(Range{Start: start, End: end}, "")
		if !changed {
			return
		}
		b.cursor = nextCursor
		b.sel = selectionState{}
		b.version++
		b.recordUndo(prev)
		return
	}

	// Join with next line (delete the newline).
	start := Pos{Row: row, Col: col}
	end := Pos{Row: row + 1, Col: 0}
	nextCursor, changed := b.replaceRange(Range{Start: start, End: end}, "")
	if !changed {
		return
	}
	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
}

func (b *Buffer) DeleteSelection() {
	r, ok := b.Selection()
	if !ok {
		return
	}
	prev := b.snapshot()
	nextCursor, changed := b.replaceRange(r, "")
	if !changed {
		return
	}
	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
}

func (b *Buffer) replaceRange(r Range, text string) (nextCursor Pos, changed bool) {
	r = NormalizeRange(ClampRange(r, len(b.lines), b.lineLen))
	if r.IsEmpty() && text == "" {
		return b.cursor, false
	}

	if r.Start.Row == r.End.Row && r.Start.Col == r.End.Col && text == "" {
		return b.cursor, false
	}

	startRow, startCol := r.Start.Row, r.Start.Col
	endRow, endCol := r.End.Row, r.End.Col

	prefix := append([]rune(nil), b.lines[startRow][:startCol]...)
	suffix := append([]rune(nil), b.lines[endRow][endCol:]...)

	parts := strings.Split(text, "\n")
	ins := make([][]rune, 0, len(parts))
	for _, p := range parts {
		ins = append(ins, []rune(p))
	}

	repl := make([][]rune, 0, len(ins))
	if len(ins) == 1 {
		line := make([]rune, 0, len(prefix)+len(ins[0])+len(suffix))
		line = append(line, prefix...)
		line = append(line, ins[0]...)
		line = append(line, suffix...)
		repl = append(repl, line)
		nextCursor = Pos{Row: startRow, Col: len(prefix) + len(ins[0])}
	} else {
		first := make([]rune, 0, len(prefix)+len(ins[0]))
		first = append(first, prefix...)
		first = append(first, ins[0]...)
		repl = append(repl, first)

		for i := 1; i < len(ins)-1; i++ {
			repl = append(repl, append([]rune(nil), ins[i]...))
		}

		lastPart := ins[len(ins)-1]
		last := make([]rune, 0, len(lastPart)+len(suffix))
		last = append(last, lastPart...)
		last = append(last, suffix...)
		repl = append(repl, last)

		nextCursor = Pos{Row: startRow + len(ins) - 1, Col: len(lastPart)}
	}

	before := b.lines[:startRow]
	after := b.lines[endRow+1:]
	out := make([][]rune, 0, len(before)+len(repl)+len(after))
	out = append(out, before...)
	out = append(out, repl...)
	out = append(out, after...)
	if len(out) == 0 {
		out = [][]rune{nil}
	}

	// No-op detection for "replace with identical text".
	// Only check in the simple single-line case to keep this cheap.
	if startRow == endRow && len(ins) == 1 {
		old := b.lines[startRow]
		if startCol <= len(old) && endCol <= len(old) {
			if string(old[startCol:endCol]) == text && len(before)+1+len(after) == len(out) {
				// Range text equals replacement and no line count change => no-op.
				return b.cursor, false
			}
		}
	}

	b.lines = out
	return nextCursor, true
}
