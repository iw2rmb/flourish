package buffer

// Pos points into the logical document by (row, col) in runes.
// Row and Col are 0-based.
type Pos struct {
	Row int
	Col int
}

// Range is a half-open selection in document coordinates: [Start, End).
// Start <= End in document order.
type Range struct {
	Start Pos
	End   Pos
}

// TextEdit replaces the text in Range with Text (which may contain '\n').
type TextEdit struct {
	Range Range
	Text  string
}

func ComparePos(a, b Pos) int {
	if a.Row < b.Row {
		return -1
	}
	if a.Row > b.Row {
		return 1
	}
	if a.Col < b.Col {
		return -1
	}
	if a.Col > b.Col {
		return 1
	}
	return 0
}

func NormalizeRange(r Range) Range {
	if ComparePos(r.Start, r.End) <= 0 {
		return r
	}
	return Range{Start: r.End, End: r.Start}
}

func (r Range) IsEmpty() bool {
	return r.Start == r.End
}

func clampInt(v, min, max int) int {
	if max < min {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ClampPos clamps p into document bounds described by rowCount and lineLen.
//
// - rowCount is the number of logical lines (rows).
// - lineLen(row) returns the rune length of the given row.
//
// The returned Pos always satisfies:
// - 0 <= Row < rowCount (with rowCount treated as at least 1)
// - 0 <= Col <= lineLen(Row)
func ClampPos(p Pos, rowCount int, lineLen func(row int) int) Pos {
	if rowCount <= 0 {
		rowCount = 1
	}

	row := clampInt(p.Row, 0, rowCount-1)

	maxCol := 0
	if lineLen != nil {
		maxCol = lineLen(row)
		if maxCol < 0 {
			maxCol = 0
		}
	}
	col := clampInt(p.Col, 0, maxCol)

	return Pos{Row: row, Col: col}
}

func ClampRange(r Range, rowCount int, lineLen func(row int) int) Range {
	return Range{
		Start: ClampPos(r.Start, rowCount, lineLen),
		End:   ClampPos(r.End, rowCount, lineLen),
	}
}
