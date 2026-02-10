package buffer

import "strings"

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
	lines   [][]rune
	version uint64

	cursor Pos
	sel    selectionState

	opt  Options
	hist historyState
}

func New(text string, opt Options) *Buffer {
	if opt.HistoryLimit == 0 {
		opt.HistoryLimit = 1000
	}
	return &Buffer{
		lines:   splitLines(text),
		version: 0,
		cursor:  Pos{Row: 0, Col: 0},
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
		sb.WriteString(string(line))
	}
	return sb.String()
}

func (b *Buffer) Version() uint64 { return b.version }

func (b *Buffer) Cursor() Pos { return b.cursor }

func (b *Buffer) SetCursor(p Pos) {
	next := b.clampPos(p)
	if next == b.cursor {
		return
	}
	b.cursor = next
	b.version++
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
		nextRange, nextOK = NormalizeRange(Range{Start: next.anchor, End: next.end}), true
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
}

func (b *Buffer) ClearSelection() {
	if !b.sel.active {
		return
	}
	if r, ok := b.Selection(); !ok || r.IsEmpty() {
		b.sel = selectionState{}
		return
	}
	b.sel = selectionState{}
	b.version++
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

func splitLines(text string) [][]rune {
	parts := strings.Split(text, "\n")
	if len(parts) == 0 {
		parts = []string{""}
	}
	lines := make([][]rune, 0, len(parts))
	for _, s := range parts {
		lines = append(lines, []rune(s))
	}
	if len(lines) == 0 {
		lines = append(lines, nil)
	}
	return lines
}
