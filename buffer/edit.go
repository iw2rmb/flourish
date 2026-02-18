package buffer

import (
	"strings"

	"github.com/iw2rmb/flourish/internal/grapheme"
)

// InsertText inserts text at the cursor, or replaces the active selection.
func (b *Buffer) InsertText(s string) {
	if s == "" {
		if _, ok := b.Selection(); ok {
			b.DeleteSelection()
		}
		return
	}

	prev := b.snapshot()
	change := b.beginChange(ChangeSourceLocal)

	r, ok := b.Selection()
	if !ok {
		r = Range{Start: b.cursor, End: b.cursor}
	}

	nextCursor, applied, changed := b.replaceRange(r, s)
	if !changed {
		return
	}

	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
	change.addAppliedEdit(applied)
	b.commitChange(change)
}

// InsertGrapheme inserts a single grapheme cluster at the cursor, or replaces
// the active selection.
func (b *Buffer) InsertGrapheme(g string) {
	if g == "" {
		return
	}
	b.InsertText(g)
}

// InsertNewline inserts a line break at the cursor, or replaces the active
// selection.
func (b *Buffer) InsertNewline() {
	b.InsertText("\n")
}

// DeleteBackward applies backspace semantics.
func (b *Buffer) DeleteBackward() {
	if _, ok := b.Selection(); ok {
		b.DeleteSelection()
		return
	}

	row, col := b.cursor.Row, b.cursor.GraphemeCol
	if row == 0 && col == 0 {
		return
	}

	prev := b.snapshot()
	change := b.beginChange(ChangeSourceLocal)

	if col > 0 {
		start := Pos{Row: row, GraphemeCol: col - 1}
		end := Pos{Row: row, GraphemeCol: col}
		nextCursor, applied, changed := b.replaceRange(Range{Start: start, End: end}, "")
		if !changed {
			return
		}
		b.cursor = nextCursor
		b.sel = selectionState{}
		b.version++
		b.recordUndo(prev)
		change.addAppliedEdit(applied)
		b.commitChange(change)
		return
	}

	// Join with previous line (delete the newline).
	prevRow := row - 1
	start := Pos{Row: prevRow, GraphemeCol: len(b.lines[prevRow])}
	end := Pos{Row: row, GraphemeCol: 0}
	nextCursor, applied, changed := b.replaceRange(Range{Start: start, End: end}, "")
	if !changed {
		return
	}
	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
	change.addAppliedEdit(applied)
	b.commitChange(change)
}

// DeleteForward applies delete-key semantics.
func (b *Buffer) DeleteForward() {
	if _, ok := b.Selection(); ok {
		b.DeleteSelection()
		return
	}

	row, col := b.cursor.Row, b.cursor.GraphemeCol
	lastRow := len(b.lines) - 1
	if row == lastRow && col == len(b.lines[lastRow]) {
		return
	}

	prev := b.snapshot()
	change := b.beginChange(ChangeSourceLocal)

	if col < len(b.lines[row]) {
		start := Pos{Row: row, GraphemeCol: col}
		end := Pos{Row: row, GraphemeCol: col + 1}
		nextCursor, applied, changed := b.replaceRange(Range{Start: start, End: end}, "")
		if !changed {
			return
		}
		b.cursor = nextCursor
		b.sel = selectionState{}
		b.version++
		b.recordUndo(prev)
		change.addAppliedEdit(applied)
		b.commitChange(change)
		return
	}

	// Join with next line (delete the newline).
	start := Pos{Row: row, GraphemeCol: col}
	end := Pos{Row: row + 1, GraphemeCol: 0}
	nextCursor, applied, changed := b.replaceRange(Range{Start: start, End: end}, "")
	if !changed {
		return
	}
	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
	change.addAppliedEdit(applied)
	b.commitChange(change)
}

// DeleteSelection deletes the active selection, if any.
func (b *Buffer) DeleteSelection() {
	r, ok := b.Selection()
	if !ok {
		return
	}
	prev := b.snapshot()
	change := b.beginChange(ChangeSourceLocal)
	nextCursor, applied, changed := b.replaceRange(r, "")
	if !changed {
		return
	}
	b.cursor = nextCursor
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
	change.addAppliedEdit(applied)
	b.commitChange(change)
}

func (b *Buffer) replaceRange(r Range, text string) (nextCursor Pos, applied AppliedEdit, changed bool) {
	r = NormalizeRange(ClampRange(r, len(b.lines), b.lineLen))
	if r.IsEmpty() && text == "" {
		return b.cursor, AppliedEdit{}, false
	}

	if r.Start.Row == r.End.Row && r.Start.GraphemeCol == r.End.GraphemeCol && text == "" {
		return b.cursor, AppliedEdit{}, false
	}

	startRow, startCol := r.Start.Row, r.Start.GraphemeCol
	endRow, endCol := r.End.Row, r.End.GraphemeCol
	deletedText := textForLinesRange(b.lines, r)
	if deletedText == text {
		return b.cursor, AppliedEdit{}, false
	}

	prefix := append([]string(nil), b.lines[startRow][:startCol]...)
	suffix := append([]string(nil), b.lines[endRow][endCol:]...)

	parts := strings.Split(text, "\n")
	ins := make([][]string, 0, len(parts))
	for _, p := range parts {
		ins = append(ins, grapheme.Split(p))
	}

	repl := make([][]string, 0, len(ins))
	if len(ins) == 1 {
		line := make([]string, 0, len(prefix)+len(ins[0])+len(suffix))
		line = append(line, prefix...)
		line = append(line, ins[0]...)
		line = append(line, suffix...)
		repl = append(repl, line)
		nextCursor = Pos{Row: startRow, GraphemeCol: len(prefix) + len(ins[0])}
	} else {
		first := make([]string, 0, len(prefix)+len(ins[0]))
		first = append(first, prefix...)
		first = append(first, ins[0]...)
		repl = append(repl, first)

		for i := 1; i < len(ins)-1; i++ {
			repl = append(repl, append([]string(nil), ins[i]...))
		}

		lastPart := ins[len(ins)-1]
		last := make([]string, 0, len(lastPart)+len(suffix))
		last = append(last, lastPart...)
		last = append(last, suffix...)
		repl = append(repl, last)

		nextCursor = Pos{Row: startRow + len(ins) - 1, GraphemeCol: len(lastPart)}
	}

	before := b.lines[:startRow]
	after := b.lines[endRow+1:]
	out := make([][]string, 0, len(before)+len(repl)+len(after))
	out = append(out, before...)
	out = append(out, repl...)
	out = append(out, after...)
	if len(out) == 0 {
		out = [][]string{nil}
	}

	b.lines = out
	applied = AppliedEdit{
		RangeBefore: r,
		RangeAfter: Range{
			Start: r.Start,
			End:   nextCursor,
		},
		InsertText:  text,
		DeletedText: deletedText,
	}
	return nextCursor, applied, true
}

func textForLinesRange(lines [][]string, r Range) string {
	r = NormalizeRange(r)
	if r.IsEmpty() {
		return ""
	}

	startRow := r.Start.Row
	endRow := r.End.Row
	startCol := r.Start.GraphemeCol
	endCol := r.End.GraphemeCol

	if startRow == endRow {
		return grapheme.Join(lines[startRow][startCol:endCol])
	}

	var sb strings.Builder
	for row := startRow; row <= endRow; row++ {
		if row > startRow {
			sb.WriteByte('\n')
		}
		partStart := 0
		partEnd := len(lines[row])
		if row == startRow {
			partStart = startCol
		}
		if row == endRow {
			partEnd = endCol
		}
		sb.WriteString(grapheme.Join(lines[row][partStart:partEnd]))
	}
	return sb.String()
}
