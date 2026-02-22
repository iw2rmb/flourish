package buffer

import (
	"unicode/utf16"
	"unicode/utf8"

	"github.com/iw2rmb/flourish/internal/grapheme"
)

type OffsetClampMode uint8

const (
	OffsetError OffsetClampMode = iota
	OffsetClamp
)

type ConvertPolicy struct {
	ClampMode OffsetClampMode
}

type GapBias uint8

const (
	GapBiasLeft GapBias = iota
	GapBiasRight
)

type Gap struct {
	RuneOffset int
	Bias       GapBias
}

type offsetUnit uint8

const (
	offsetUnitByte offsetUnit = iota
	offsetUnitRune
	offsetUnitUTF16
)

func (b *Buffer) PosFromByteOffset(off int, p ConvertPolicy) (Pos, bool) {
	off, ok := clampOffset(off, b.docLen(offsetUnitByte), p.ClampMode)
	if !ok {
		return Pos{}, false
	}
	return b.posFromOffset(off, offsetUnitByte)
}

func (b *Buffer) ByteOffsetFromPos(pos Pos, p ConvertPolicy) (int, bool) {
	pos, ok := b.normalizePosForMode(pos, p.ClampMode)
	if !ok {
		return 0, false
	}
	return b.offsetFromPos(pos, offsetUnitByte), true
}

func (b *Buffer) PosFromRuneOffset(off int, p ConvertPolicy) (Pos, bool) {
	off, ok := clampOffset(off, b.docLen(offsetUnitRune), p.ClampMode)
	if !ok {
		return Pos{}, false
	}
	return b.posFromOffset(off, offsetUnitRune)
}

func (b *Buffer) RuneOffsetFromPos(pos Pos, p ConvertPolicy) (int, bool) {
	pos, ok := b.normalizePosForMode(pos, p.ClampMode)
	if !ok {
		return 0, false
	}
	return b.offsetFromPos(pos, offsetUnitRune), true
}

func (b *Buffer) PosFromUTF16Offset(off int, p ConvertPolicy) (Pos, bool) {
	off, ok := clampOffset(off, b.docLen(offsetUnitUTF16), p.ClampMode)
	if !ok {
		return Pos{}, false
	}
	return b.posFromOffset(off, offsetUnitUTF16)
}

func (b *Buffer) UTF16OffsetFromPos(pos Pos, p ConvertPolicy) (int, bool) {
	pos, ok := b.normalizePosForMode(pos, p.ClampMode)
	if !ok {
		return 0, false
	}
	return b.offsetFromPos(pos, offsetUnitUTF16), true
}

func (b *Buffer) GapFromPos(pos Pos, bias GapBias) (Gap, bool) {
	if !validGapBias(bias) {
		return Gap{}, false
	}
	off, ok := b.RuneOffsetFromPos(pos, ConvertPolicy{
		ClampMode: OffsetError,
	})
	if !ok {
		return Gap{}, false
	}
	return Gap{RuneOffset: off, Bias: bias}, true
}

func (b *Buffer) PosFromGap(g Gap, p ConvertPolicy) (Pos, bool) {
	if !validGapBias(g.Bias) {
		return Pos{}, false
	}
	return b.PosFromRuneOffset(g.RuneOffset, p)
}

func validGapBias(bias GapBias) bool {
	return bias == GapBiasLeft || bias == GapBiasRight
}

func clampOffset(off, max int, mode OffsetClampMode) (int, bool) {
	switch mode {
	case OffsetError:
		if off < 0 || off > max {
			return 0, false
		}
		return off, true
	case OffsetClamp:
		if off < 0 {
			return 0, true
		}
		if off > max {
			return max, true
		}
		return off, true
	default:
		return 0, false
	}
}

func (b *Buffer) normalizePosForMode(pos Pos, mode OffsetClampMode) (Pos, bool) {
	switch mode {
	case OffsetError:
		clamped := b.clampPos(pos)
		if clamped != pos {
			return Pos{}, false
		}
		return pos, true
	case OffsetClamp:
		return b.clampPos(pos), true
	default:
		return Pos{}, false
	}
}

func unitWidth(cluster string, unit offsetUnit) int {
	if unit == offsetUnitRune {
		return utf8.RuneCountInString(cluster)
	}
	if unit == offsetUnitUTF16 {
		width := 0
		for _, r := range cluster {
			n := utf16.RuneLen(r)
			if n < 0 {
				n = 1
			}
			width += n
		}
		return width
	}
	return len(cluster)
}

func (b *Buffer) ensureOffsetIndex() {
	if b.offsetIdx.valid && b.offsetIdx.textVersion == b.textVersion {
		return
	}
	n := len(b.lines)
	if cap(b.offsetIdx.byteStarts) >= n {
		b.offsetIdx.byteStarts = b.offsetIdx.byteStarts[:n]
		b.offsetIdx.runeStarts = b.offsetIdx.runeStarts[:n]
		b.offsetIdx.utf16Starts = b.offsetIdx.utf16Starts[:n]
	} else {
		b.offsetIdx.byteStarts = make([]int, n)
		b.offsetIdx.runeStarts = make([]int, n)
		b.offsetIdx.utf16Starts = make([]int, n)
	}

	byteOff, runeOff, utf16Off := 0, 0, 0
	for i, line := range b.lines {
		b.offsetIdx.byteStarts[i] = byteOff
		b.offsetIdx.runeStarts[i] = runeOff
		b.offsetIdx.utf16Starts[i] = utf16Off
		for _, cluster := range line {
			byteOff += len(cluster)
			runeOff += utf8.RuneCountInString(cluster)
			for _, r := range cluster {
				n := utf16.RuneLen(r)
				if n < 0 {
					n = 1
				}
				utf16Off += n
			}
		}
		if i < len(b.lines)-1 {
			byteOff++
			runeOff++
			utf16Off++
		}
	}
	b.offsetIdx.textVersion = b.textVersion
	b.offsetIdx.valid = true
}

func (b *Buffer) lineStarts(unit offsetUnit) []int {
	b.ensureOffsetIndex()
	switch unit {
	case offsetUnitRune:
		return b.offsetIdx.runeStarts
	case offsetUnitUTF16:
		return b.offsetIdx.utf16Starts
	default:
		return b.offsetIdx.byteStarts
	}
}

func (b *Buffer) docLen(unit offsetUnit) int {
	starts := b.lineStarts(unit)
	n := len(b.lines)
	if n == 0 {
		return 0
	}
	// Total = start of last line + width of last line content.
	lastStart := starts[n-1]
	w := 0
	for _, cluster := range b.lines[n-1] {
		w += unitWidth(cluster, unit)
	}
	return lastStart + w
}

func (b *Buffer) posFromOffset(off int, unit offsetUnit) (Pos, bool) {
	starts := b.lineStarts(unit)
	n := len(b.lines)
	if n == 0 {
		return Pos{}, false
	}

	// Binary search for the row.
	lo, hi := 0, n-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if starts[mid] <= off {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	row := lo

	// Linear scan within the line to find the column.
	cur := starts[row]
	if off == cur {
		return Pos{Row: row, GraphemeCol: 0}, true
	}

	for col, cluster := range b.lines[row] {
		next := cur + unitWidth(cluster, unit)
		if off > cur && off < next {
			return Pos{}, false
		}
		cur = next
		if off == cur {
			return Pos{Row: row, GraphemeCol: col + 1}, true
		}
	}

	// Check if offset points to the newline between this row and the next.
	if row < n-1 {
		cur++ // newline
		if off == cur {
			return Pos{Row: row + 1, GraphemeCol: 0}, true
		}
	}

	return Pos{}, false
}

func (b *Buffer) offsetFromPos(pos Pos, unit offsetUnit) int {
	starts := b.lineStarts(unit)
	off := starts[pos.Row]

	for col := 0; col < pos.GraphemeCol; col++ {
		off += unitWidth(b.lines[pos.Row][col], unit)
	}

	return off
}

func GraphemeColFromRuneOffsetInLine(line string, runeOff int, clamp OffsetClampMode) (int, bool) {
	clusters := grapheme.Split(line)
	totalRunes := 0
	for _, cluster := range clusters {
		totalRunes += utf8.RuneCountInString(cluster)
	}

	off, ok := clampOffset(runeOff, totalRunes, clamp)
	if !ok {
		return 0, false
	}

	cur := 0
	for col, cluster := range clusters {
		if off == cur {
			return col, true
		}
		next := cur + utf8.RuneCountInString(cluster)
		if off > cur && off < next {
			return 0, false
		}
		cur = next
	}

	if off == cur {
		return len(clusters), true
	}
	return 0, false
}

func RuneOffsetFromGraphemeColInLine(line string, graphemeCol int, clamp OffsetClampMode) (int, bool) {
	clusters := grapheme.Split(line)

	col, ok := clampOffset(graphemeCol, len(clusters), clamp)
	if !ok {
		return 0, false
	}

	off := 0
	for i := 0; i < col; i++ {
		off += utf8.RuneCountInString(clusters[i])
	}
	return off, true
}
