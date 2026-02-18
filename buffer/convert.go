package buffer

import "unicode/utf8"

type OffsetClampMode uint8

const (
	OffsetError OffsetClampMode = iota
	OffsetClamp
)

type NewlineMode uint8

const (
	NewlineAsSingleRune NewlineMode = iota
)

type ConvertPolicy struct {
	ClampMode   OffsetClampMode
	NewlineMode NewlineMode
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
)

func (b *Buffer) PosFromByteOffset(off int, p ConvertPolicy) (Pos, bool) {
	if !validNewlineMode(p.NewlineMode) {
		return Pos{}, false
	}

	off, ok := clampOffset(off, b.docLen(offsetUnitByte), p.ClampMode)
	if !ok {
		return Pos{}, false
	}
	return b.posFromOffset(off, offsetUnitByte)
}

func (b *Buffer) ByteOffsetFromPos(pos Pos, p ConvertPolicy) (int, bool) {
	if !validNewlineMode(p.NewlineMode) {
		return 0, false
	}

	pos, ok := b.normalizePosForMode(pos, p.ClampMode)
	if !ok {
		return 0, false
	}
	return b.offsetFromPos(pos, offsetUnitByte), true
}

func (b *Buffer) PosFromRuneOffset(off int, p ConvertPolicy) (Pos, bool) {
	if !validNewlineMode(p.NewlineMode) {
		return Pos{}, false
	}

	off, ok := clampOffset(off, b.docLen(offsetUnitRune), p.ClampMode)
	if !ok {
		return Pos{}, false
	}
	return b.posFromOffset(off, offsetUnitRune)
}

func (b *Buffer) RuneOffsetFromPos(pos Pos, p ConvertPolicy) (int, bool) {
	if !validNewlineMode(p.NewlineMode) {
		return 0, false
	}

	pos, ok := b.normalizePosForMode(pos, p.ClampMode)
	if !ok {
		return 0, false
	}
	return b.offsetFromPos(pos, offsetUnitRune), true
}

func (b *Buffer) GapFromPos(pos Pos, bias GapBias) (Gap, bool) {
	if !validGapBias(bias) {
		return Gap{}, false
	}
	off, ok := b.RuneOffsetFromPos(pos, ConvertPolicy{
		ClampMode:   OffsetError,
		NewlineMode: NewlineAsSingleRune,
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

func validNewlineMode(mode NewlineMode) bool {
	return mode == NewlineAsSingleRune
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
	return len(cluster)
}

func (b *Buffer) docLen(unit offsetUnit) int {
	total := 0
	for row, line := range b.lines {
		for _, cluster := range line {
			total += unitWidth(cluster, unit)
		}
		if row < len(b.lines)-1 {
			total++
		}
	}
	return total
}

func (b *Buffer) posFromOffset(off int, unit offsetUnit) (Pos, bool) {
	cur := 0

	for row, line := range b.lines {
		col := 0
		if off == cur {
			return Pos{Row: row, GraphemeCol: col}, true
		}

		for _, cluster := range line {
			next := cur + unitWidth(cluster, unit)
			if off > cur && off < next {
				return Pos{}, false
			}
			cur = next
			col++
			if off == cur {
				return Pos{Row: row, GraphemeCol: col}, true
			}
		}

		if row < len(b.lines)-1 {
			cur++
			if off == cur {
				return Pos{Row: row + 1, GraphemeCol: 0}, true
			}
		}
	}

	return Pos{}, false
}

func (b *Buffer) offsetFromPos(pos Pos, unit offsetUnit) int {
	off := 0

	for row := 0; row < pos.Row; row++ {
		for _, cluster := range b.lines[row] {
			off += unitWidth(cluster, unit)
		}
		off++
	}

	for col := 0; col < pos.GraphemeCol; col++ {
		off += unitWidth(b.lines[pos.Row][col], unit)
	}

	return off
}
