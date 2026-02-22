package buffer

import (
	"strings"

	"github.com/iw2rmb/flourish/internal/grapheme"
)

type Options struct {
	HistoryLimit int // default: 1000 (wired in later phases)
}

type selectionState struct {
	active bool
	anchor Pos
	end    Pos
}

// Buffer is the pure document state: text, cursor, and selection.
type Buffer struct {
	lines   [][]string
	version uint64
	// textVersion advances only when document text changes.
	textVersion uint64

	cursor Pos
	sel    selectionState

	lastChange    Change
	hasLastChange bool

	opt  Options
	hist historyState

	offsetIdx lineOffsetIndex
}

// lineOffsetIndex caches cumulative byte/rune/UTF-16 offsets per line so that
// offset ↔ position conversions are O(log n + line length) instead of O(document).
type lineOffsetIndex struct {
	textVersion uint64
	valid       bool
	// lineStart[unit][i] = document offset at the start of line i.
	// Length: len(lines).
	byteStarts  []int
	runeStarts  []int
	utf16Starts []int
}

func New(text string, opt Options) *Buffer {
	if opt.HistoryLimit == 0 {
		opt.HistoryLimit = 1000
	}
	return &Buffer{
		lines:   splitLines(text),
		version: 0,
		cursor:  Pos{Row: 0, GraphemeCol: 0},
		sel:     selectionState{},
		opt:     opt,
	}
}

func (b *Buffer) Text() string {
	if len(b.lines) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, line := range b.lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(grapheme.Join(line))
	}
	return sb.String()
}

func (b *Buffer) Version() uint64 { return b.version }

// TextVersion increments only on effective text mutations.
//
// Cursor/selection-only changes do not change TextVersion.
func (b *Buffer) TextVersion() uint64 { return b.textVersion }

func (b *Buffer) Cursor() Pos { return b.cursor }

func (b *Buffer) SetCursor(p Pos) {
	next := b.clampPos(p)
	if next == b.cursor {
		return
	}
	change := b.beginChange(ChangeSourceLocal)
	b.cursor = next
	b.version++
	b.commitChange(change)
}

func (b *Buffer) Selection() (Range, bool) {
	if !b.sel.active {
		return Range{}, false
	}
	r := NormalizeRange(Range{Start: b.sel.anchor, End: b.sel.end})
	if r.IsEmpty() {
		return Range{}, false
	}
	return r, true
}

// SelectionRaw returns the raw selection anchor/end without normalization.
//
// This is useful for UI layers that need to preserve the selection direction
// (e.g. shift+click behavior) while still treating empty selections as inactive.
func (b *Buffer) SelectionRaw() (Range, bool) {
	if !b.sel.active || b.sel.anchor == b.sel.end {
		return Range{}, false
	}
	return Range{Start: b.sel.anchor, End: b.sel.end}, true
}

func (b *Buffer) SetSelection(r Range) {
	change := b.beginChange(ChangeSourceLocal)

	clamped := ClampRange(r, len(b.lines), b.lineLen)
	next := selectionState{
		active: true,
		anchor: clamped.Start,
		end:    clamped.End,
	}
	norm := NormalizeRange(Range{Start: next.anchor, End: next.end})
	if norm.IsEmpty() {
		next = selectionState{}
	}

	prevRange, prevOK := b.Selection()
	nextRange, nextOK := Range{}, false
	if next.active {
		nextRange, nextOK = norm, true
		if nextRange.IsEmpty() {
			nextRange, nextOK = Range{}, false
		}
	}

	if prevOK == nextOK && (!prevOK || prevRange == nextRange) {
		b.sel = next
		return
	}

	b.sel = next
	b.version++
	b.commitChange(change)
}

func (b *Buffer) ClearSelection() {
	if !b.sel.active {
		return
	}
	change := b.beginChange(ChangeSourceLocal)
	if r, ok := b.Selection(); !ok || r.IsEmpty() {
		b.sel = selectionState{}
		return
	}
	b.sel = selectionState{}
	b.version++
	b.commitChange(change)
}

func (b *Buffer) lineLen(row int) int {
	if row < 0 || row >= len(b.lines) {
		return 0
	}
	return len(b.lines[row])
}

func (b *Buffer) clampPos(p Pos) Pos {
	return ClampPos(p, len(b.lines), b.lineLen)
}

// TextInRange returns the text contained in the given range, reading directly
// from the internal grapheme-split line storage. This avoids the full document
// serialization that Text() + strings.Split would require.
func (b *Buffer) TextInRange(r Range) string {
	r = NormalizeRange(r)
	if r.IsEmpty() {
		return ""
	}
	if r.Start.Row < 0 || r.Start.Row >= len(b.lines) || r.End.Row < 0 || r.End.Row >= len(b.lines) {
		return ""
	}

	if r.Start.Row == r.End.Row {
		line := b.lines[r.Start.Row]
		startCol := r.Start.GraphemeCol
		endCol := r.End.GraphemeCol
		if startCol < 0 || endCol < 0 || startCol > len(line) || endCol > len(line) {
			return ""
		}
		return grapheme.Join(line[startCol:endCol])
	}

	var sb strings.Builder
	for row := r.Start.Row; row <= r.End.Row; row++ {
		if row > r.Start.Row {
			sb.WriteByte('\n')
		}
		line := b.lines[row]
		startCol := 0
		endCol := len(line)
		if row == r.Start.Row {
			startCol = r.Start.GraphemeCol
		}
		if row == r.End.Row {
			endCol = r.End.GraphemeCol
		}
		if startCol < 0 || endCol < 0 || startCol > len(line) || endCol > len(line) || startCol > endCol {
			return ""
		}
		sb.WriteString(grapheme.Join(line[startCol:endCol]))
	}
	return sb.String()
}

// LineCount returns the number of lines in the buffer.
func (b *Buffer) LineCount() int { return len(b.lines) }

func splitLines(text string) [][]string {
	parts := strings.Split(text, "\n")
	lines := make([][]string, 0, len(parts))
	for _, s := range parts {
		lines = append(lines, grapheme.Split(s))
	}
	return lines
}
