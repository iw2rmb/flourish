package buffer

import "testing"

func TestBuffer_ConversionAPIs_ExportedAndCallable(t *testing.T) {
	var (
		_ func(*Buffer, int, ConvertPolicy) (Pos, bool)  = (*Buffer).PosFromByteOffset
		_ func(*Buffer, Pos, ConvertPolicy) (int, bool)  = (*Buffer).ByteOffsetFromPos
		_ func(*Buffer, int, ConvertPolicy) (Pos, bool)  = (*Buffer).PosFromRuneOffset
		_ func(*Buffer, Pos, ConvertPolicy) (int, bool)  = (*Buffer).RuneOffsetFromPos
		_ func(*Buffer, int, ConvertPolicy) (Pos, bool)  = (*Buffer).PosFromUTF16Offset
		_ func(*Buffer, Pos, ConvertPolicy) (int, bool)  = (*Buffer).UTF16OffsetFromPos
		_ func(*Buffer, Pos, GapBias) (Gap, bool)        = (*Buffer).GapFromPos
		_ func(*Buffer, Gap, ConvertPolicy) (Pos, bool)  = (*Buffer).PosFromGap
		_ func(string, int, OffsetClampMode) (int, bool) = GraphemeColFromRuneOffsetInLine
		_ func(string, int, OffsetClampMode) (int, bool) = RuneOffsetFromGraphemeColInLine
	)

	b := New("ab\ncd", Options{})
	policy := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}
	if _, ok := b.PosFromByteOffset(0, policy); !ok {
		t.Fatalf("PosFromByteOffset should be callable")
	}
	if _, ok := b.ByteOffsetFromPos(Pos{Row: 0, GraphemeCol: 0}, policy); !ok {
		t.Fatalf("ByteOffsetFromPos should be callable")
	}
	if _, ok := b.PosFromRuneOffset(0, policy); !ok {
		t.Fatalf("PosFromRuneOffset should be callable")
	}
	if _, ok := b.RuneOffsetFromPos(Pos{Row: 0, GraphemeCol: 0}, policy); !ok {
		t.Fatalf("RuneOffsetFromPos should be callable")
	}
	if _, ok := b.PosFromUTF16Offset(0, policy); !ok {
		t.Fatalf("PosFromUTF16Offset should be callable")
	}
	if _, ok := b.UTF16OffsetFromPos(Pos{Row: 0, GraphemeCol: 0}, policy); !ok {
		t.Fatalf("UTF16OffsetFromPos should be callable")
	}
	if _, ok := b.GapFromPos(Pos{Row: 0, GraphemeCol: 0}, GapBiasLeft); !ok {
		t.Fatalf("GapFromPos should be callable")
	}
	if _, ok := b.PosFromGap(Gap{RuneOffset: 0, Bias: GapBiasLeft}, policy); !ok {
		t.Fatalf("PosFromGap should be callable")
	}
	if _, ok := GraphemeColFromRuneOffsetInLine("a", 0, OffsetError); !ok {
		t.Fatalf("GraphemeColFromRuneOffsetInLine should be callable")
	}
	if _, ok := RuneOffsetFromGraphemeColInLine("a", 0, OffsetError); !ok {
		t.Fatalf("RuneOffsetFromGraphemeColInLine should be callable")
	}
}

func TestBuffer_PosFromByteOffset(t *testing.T) {
	b := New("ab\ncd", Options{})

	clamp := ConvertPolicy{ClampMode: OffsetClamp, NewlineMode: NewlineAsSingleRune}
	errMode := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}

	cases := []struct {
		name string
		off  int
		p    ConvertPolicy
		want Pos
		ok   bool
	}{
		{name: "bof", off: 0, p: errMode, want: Pos{Row: 0, GraphemeCol: 0}, ok: true},
		{name: "line-0-middle", off: 1, p: errMode, want: Pos{Row: 0, GraphemeCol: 1}, ok: true},
		{name: "line-0-end", off: 2, p: errMode, want: Pos{Row: 0, GraphemeCol: 2}, ok: true},
		{name: "newline-after", off: 3, p: errMode, want: Pos{Row: 1, GraphemeCol: 0}, ok: true},
		{name: "eof", off: 5, p: errMode, want: Pos{Row: 1, GraphemeCol: 2}, ok: true},
		{name: "below-range-error", off: -1, p: errMode, ok: false},
		{name: "above-range-error", off: 6, p: errMode, ok: false},
		{name: "below-range-clamp", off: -1, p: clamp, want: Pos{Row: 0, GraphemeCol: 0}, ok: true},
		{name: "above-range-clamp", off: 6, p: clamp, want: Pos{Row: 1, GraphemeCol: 2}, ok: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := b.PosFromByteOffset(tc.off, tc.p)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("pos=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuffer_PosFromByteOffset_RejectsInteriorClusterBytes(t *testing.T) {
	b := New("éx", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}
	if _, ok := b.PosFromByteOffset(1, p); ok {
		t.Fatalf("expected byte offset inside grapheme cluster to fail")
	}
}

func TestBuffer_PosFromRuneOffset(t *testing.T) {
	b := New("ab\ncd", Options{})

	clamp := ConvertPolicy{ClampMode: OffsetClamp, NewlineMode: NewlineAsSingleRune}
	errMode := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}

	cases := []struct {
		name string
		off  int
		p    ConvertPolicy
		want Pos
		ok   bool
	}{
		{name: "bof", off: 0, p: errMode, want: Pos{Row: 0, GraphemeCol: 0}, ok: true},
		{name: "line-0-middle", off: 1, p: errMode, want: Pos{Row: 0, GraphemeCol: 1}, ok: true},
		{name: "line-0-end", off: 2, p: errMode, want: Pos{Row: 0, GraphemeCol: 2}, ok: true},
		{name: "newline-after", off: 3, p: errMode, want: Pos{Row: 1, GraphemeCol: 0}, ok: true},
		{name: "eof", off: 5, p: errMode, want: Pos{Row: 1, GraphemeCol: 2}, ok: true},
		{name: "below-range-error", off: -1, p: errMode, ok: false},
		{name: "above-range-error", off: 6, p: errMode, ok: false},
		{name: "below-range-clamp", off: -1, p: clamp, want: Pos{Row: 0, GraphemeCol: 0}, ok: true},
		{name: "above-range-clamp", off: 6, p: clamp, want: Pos{Row: 1, GraphemeCol: 2}, ok: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := b.PosFromRuneOffset(tc.off, tc.p)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("pos=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuffer_PosFromRuneOffset_RejectsInteriorClusterRunes(t *testing.T) {
	b := New("e\u0301x", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}
	if _, ok := b.PosFromRuneOffset(1, p); ok {
		t.Fatalf("expected rune offset inside grapheme cluster to fail")
	}
}

func TestBuffer_PosFromUTF16Offset(t *testing.T) {
	b := New("ab\n😀", Options{})

	clamp := ConvertPolicy{ClampMode: OffsetClamp, NewlineMode: NewlineAsSingleRune}
	errMode := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}

	cases := []struct {
		name string
		off  int
		p    ConvertPolicy
		want Pos
		ok   bool
	}{
		{name: "bof", off: 0, p: errMode, want: Pos{Row: 0, GraphemeCol: 0}, ok: true},
		{name: "line-0-middle", off: 1, p: errMode, want: Pos{Row: 0, GraphemeCol: 1}, ok: true},
		{name: "line-0-end", off: 2, p: errMode, want: Pos{Row: 0, GraphemeCol: 2}, ok: true},
		{name: "newline-after", off: 3, p: errMode, want: Pos{Row: 1, GraphemeCol: 0}, ok: true},
		{name: "eof", off: 5, p: errMode, want: Pos{Row: 1, GraphemeCol: 1}, ok: true},
		{name: "below-range-error", off: -1, p: errMode, ok: false},
		{name: "above-range-error", off: 6, p: errMode, ok: false},
		{name: "below-range-clamp", off: -1, p: clamp, want: Pos{Row: 0, GraphemeCol: 0}, ok: true},
		{name: "above-range-clamp", off: 6, p: clamp, want: Pos{Row: 1, GraphemeCol: 1}, ok: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := b.PosFromUTF16Offset(tc.off, tc.p)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("pos=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuffer_PosFromUTF16Offset_RejectsInteriorClusterUnits(t *testing.T) {
	b := New("😀x", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}
	if _, ok := b.PosFromUTF16Offset(1, p); ok {
		t.Fatalf("expected utf16 offset inside grapheme cluster to fail")
	}
}

func TestBuffer_OffsetFromPos(t *testing.T) {
	b := New("ab\né", Options{})
	errMode := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}
	clamp := ConvertPolicy{ClampMode: OffsetClamp, NewlineMode: NewlineAsSingleRune}

	gotByte, ok := b.ByteOffsetFromPos(Pos{Row: 1, GraphemeCol: 1}, errMode)
	if !ok || gotByte != 5 {
		t.Fatalf("ByteOffsetFromPos=(%d,%v), want (5,true)", gotByte, ok)
	}

	gotRune, ok := b.RuneOffsetFromPos(Pos{Row: 1, GraphemeCol: 1}, errMode)
	if !ok || gotRune != 4 {
		t.Fatalf("RuneOffsetFromPos=(%d,%v), want (4,true)", gotRune, ok)
	}
	gotUTF16, ok := b.UTF16OffsetFromPos(Pos{Row: 1, GraphemeCol: 1}, errMode)
	if !ok || gotUTF16 != 4 {
		t.Fatalf("UTF16OffsetFromPos=(%d,%v), want (4,true)", gotUTF16, ok)
	}

	if _, ok := b.ByteOffsetFromPos(Pos{Row: 99, GraphemeCol: 99}, errMode); ok {
		t.Fatalf("expected invalid pos in error mode to fail for byte conversion")
	}
	if _, ok := b.RuneOffsetFromPos(Pos{Row: 99, GraphemeCol: 99}, errMode); ok {
		t.Fatalf("expected invalid pos in error mode to fail for rune conversion")
	}
	if _, ok := b.UTF16OffsetFromPos(Pos{Row: 99, GraphemeCol: 99}, errMode); ok {
		t.Fatalf("expected invalid pos in error mode to fail for utf16 conversion")
	}

	gotByte, ok = b.ByteOffsetFromPos(Pos{Row: 99, GraphemeCol: 99}, clamp)
	if !ok || gotByte != 5 {
		t.Fatalf("ByteOffsetFromPos clamp=(%d,%v), want (5,true)", gotByte, ok)
	}
	gotRune, ok = b.RuneOffsetFromPos(Pos{Row: 99, GraphemeCol: 99}, clamp)
	if !ok || gotRune != 4 {
		t.Fatalf("RuneOffsetFromPos clamp=(%d,%v), want (4,true)", gotRune, ok)
	}
	gotUTF16, ok = b.UTF16OffsetFromPos(Pos{Row: 99, GraphemeCol: 99}, clamp)
	if !ok || gotUTF16 != 4 {
		t.Fatalf("UTF16OffsetFromPos clamp=(%d,%v), want (4,true)", gotUTF16, ok)
	}
}

func TestBuffer_GapConversions(t *testing.T) {
	b := New("a\nβ", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}

	g, ok := b.GapFromPos(Pos{Row: 1, GraphemeCol: 1}, GapBiasRight)
	if !ok {
		t.Fatalf("GapFromPos failed")
	}
	wantGap := Gap{RuneOffset: 3, Bias: GapBiasRight}
	if g != wantGap {
		t.Fatalf("gap=%v, want %v", g, wantGap)
	}

	pos, ok := b.PosFromGap(g, p)
	if !ok || pos != (Pos{Row: 1, GraphemeCol: 1}) {
		t.Fatalf("PosFromGap=(%v,%v), want ((1,1),true)", pos, ok)
	}

	if _, ok := b.GapFromPos(Pos{Row: 0, GraphemeCol: 0}, GapBias(99)); ok {
		t.Fatalf("expected invalid gap bias to fail")
	}
	if _, ok := b.PosFromGap(Gap{RuneOffset: 0, Bias: GapBias(99)}, p); ok {
		t.Fatalf("expected invalid gap bias to fail")
	}

	if _, ok := b.PosFromGap(Gap{RuneOffset: 99, Bias: GapBiasLeft}, p); ok {
		t.Fatalf("expected out-of-range gap rune offset in error mode to fail")
	}
	clamp := ConvertPolicy{ClampMode: OffsetClamp, NewlineMode: NewlineAsSingleRune}
	if got, ok := b.PosFromGap(Gap{RuneOffset: 99, Bias: GapBiasLeft}, clamp); !ok || got != (Pos{Row: 1, GraphemeCol: 1}) {
		t.Fatalf("clamped PosFromGap=(%v,%v), want ((1,1),true)", got, ok)
	}
}

func TestBuffer_OffsetConversions_DeterministicAcrossCalls(t *testing.T) {
	b := New("aé\ne\u0301\n👨‍👩‍👧‍👦", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}

	assertDeterministic := func(start, end int, fn func(int) (Pos, bool)) {
		for off := start; off <= end; off++ {
			wantPos, wantOK := fn(off)
			for i := 0; i < 8; i++ {
				gotPos, gotOK := fn(off)
				if gotOK != wantOK || gotPos != wantPos {
					t.Fatalf("off=%d call=%d got=(%v,%v) want=(%v,%v)", off, i, gotPos, gotOK, wantPos, wantOK)
				}
			}
		}
	}

	assertDeterministic(-1, b.docLen(offsetUnitByte)+1, func(off int) (Pos, bool) {
		return b.PosFromByteOffset(off, p)
	})
	assertDeterministic(-1, b.docLen(offsetUnitRune)+1, func(off int) (Pos, bool) {
		return b.PosFromRuneOffset(off, p)
	})
	assertDeterministic(-1, b.docLen(offsetUnitUTF16)+1, func(off int) (Pos, bool) {
		return b.PosFromUTF16Offset(off, p)
	})
}

func TestBuffer_OffsetConversions_RoundTripAtBoundaries(t *testing.T) {
	b := New("é\ne\u0301\n👨‍👩‍👧‍👦", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}

	cases := []Pos{
		{Row: 0, GraphemeCol: 0},
		{Row: 0, GraphemeCol: 1},
		{Row: 1, GraphemeCol: 0},
		{Row: 1, GraphemeCol: 1},
		{Row: 2, GraphemeCol: 0},
		{Row: 2, GraphemeCol: 1},
	}

	for _, pos := range cases {
		byteOff, ok := b.ByteOffsetFromPos(pos, p)
		if !ok {
			t.Fatalf("ByteOffsetFromPos(%v) failed", pos)
		}
		gotPos, ok := b.PosFromByteOffset(byteOff, p)
		if !ok || gotPos != pos {
			t.Fatalf("byte round-trip pos=%v got=(%v,%v)", pos, gotPos, ok)
		}

		runeOff, ok := b.RuneOffsetFromPos(pos, p)
		if !ok {
			t.Fatalf("RuneOffsetFromPos(%v) failed", pos)
		}
		gotPos, ok = b.PosFromRuneOffset(runeOff, p)
		if !ok || gotPos != pos {
			t.Fatalf("rune round-trip pos=%v got=(%v,%v)", pos, gotPos, ok)
		}

		utf16Off, ok := b.UTF16OffsetFromPos(pos, p)
		if !ok {
			t.Fatalf("UTF16OffsetFromPos(%v) failed", pos)
		}
		gotPos, ok = b.PosFromUTF16Offset(utf16Off, p)
		if !ok || gotPos != pos {
			t.Fatalf("utf16 round-trip pos=%v got=(%v,%v)", pos, gotPos, ok)
		}
	}
}

func TestBuffer_GapConversions_Boundaries(t *testing.T) {
	b := New("a\nβ", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}

	cases := []struct {
		pos      Pos
		wantRune int
	}{
		{pos: Pos{Row: 0, GraphemeCol: 0}, wantRune: 0},
		{pos: Pos{Row: 0, GraphemeCol: 1}, wantRune: 1},
		{pos: Pos{Row: 1, GraphemeCol: 0}, wantRune: 2},
		{pos: Pos{Row: 1, GraphemeCol: 1}, wantRune: 3},
	}
	biases := []GapBias{GapBiasLeft, GapBiasRight}

	for _, tc := range cases {
		for _, bias := range biases {
			g, ok := b.GapFromPos(tc.pos, bias)
			if !ok {
				t.Fatalf("GapFromPos(%v,%v) failed", tc.pos, bias)
			}
			wantGap := Gap{RuneOffset: tc.wantRune, Bias: bias}
			if g != wantGap {
				t.Fatalf("GapFromPos(%v,%v)=%v want=%v", tc.pos, bias, g, wantGap)
			}

			gotPos, ok := b.PosFromGap(g, p)
			if !ok || gotPos != tc.pos {
				t.Fatalf("PosFromGap(%v)=(%v,%v), want (%v,true)", g, gotPos, ok, tc.pos)
			}
		}
	}
}

func TestBuffer_ConversionAPIs_InvalidNewlineMode(t *testing.T) {
	b := New("ab", Options{})
	p := ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineMode(99)}

	if _, ok := b.PosFromByteOffset(0, p); ok {
		t.Fatalf("expected invalid newline mode to fail")
	}
	if _, ok := b.ByteOffsetFromPos(Pos{Row: 0, GraphemeCol: 0}, p); ok {
		t.Fatalf("expected invalid newline mode to fail")
	}
	if _, ok := b.PosFromRuneOffset(0, p); ok {
		t.Fatalf("expected invalid newline mode to fail")
	}
	if _, ok := b.RuneOffsetFromPos(Pos{Row: 0, GraphemeCol: 0}, p); ok {
		t.Fatalf("expected invalid newline mode to fail")
	}
	if _, ok := b.PosFromUTF16Offset(0, p); ok {
		t.Fatalf("expected invalid newline mode to fail")
	}
	if _, ok := b.UTF16OffsetFromPos(Pos{Row: 0, GraphemeCol: 0}, p); ok {
		t.Fatalf("expected invalid newline mode to fail")
	}
}

func TestLineConversionHelpers_RuneOffsetToGraphemeCol(t *testing.T) {
	line := "aée\u0301👨‍👩‍👧‍👦"

	cases := []struct {
		name string
		off  int
		mode OffsetClampMode
		want int
		ok   bool
	}{
		{name: "start", off: 0, mode: OffsetError, want: 0, ok: true},
		{name: "after-ascii", off: 1, mode: OffsetError, want: 1, ok: true},
		{name: "after-acute", off: 2, mode: OffsetError, want: 2, ok: true},
		{name: "after-combining-cluster", off: 4, mode: OffsetError, want: 3, ok: true},
		{name: "end", off: 11, mode: OffsetError, want: 4, ok: true},
		{name: "interior-combining", off: 3, mode: OffsetError, ok: false},
		{name: "interior-zwj", off: 6, mode: OffsetError, ok: false},
		{name: "below-range-error", off: -1, mode: OffsetError, ok: false},
		{name: "above-range-error", off: 12, mode: OffsetError, ok: false},
		{name: "below-range-clamp", off: -1, mode: OffsetClamp, want: 0, ok: true},
		{name: "above-range-clamp", off: 12, mode: OffsetClamp, want: 4, ok: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := GraphemeColFromRuneOffsetInLine(line, tc.off, tc.mode)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("col=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestLineConversionHelpers_GraphemeColToRuneOffset(t *testing.T) {
	line := "aée\u0301👨‍👩‍👧‍👦"

	cases := []struct {
		name string
		col  int
		mode OffsetClampMode
		want int
		ok   bool
	}{
		{name: "start", col: 0, mode: OffsetError, want: 0, ok: true},
		{name: "after-ascii", col: 1, mode: OffsetError, want: 1, ok: true},
		{name: "after-acute", col: 2, mode: OffsetError, want: 2, ok: true},
		{name: "after-combining-cluster", col: 3, mode: OffsetError, want: 4, ok: true},
		{name: "end", col: 4, mode: OffsetError, want: 11, ok: true},
		{name: "below-range-error", col: -1, mode: OffsetError, ok: false},
		{name: "above-range-error", col: 5, mode: OffsetError, ok: false},
		{name: "below-range-clamp", col: -1, mode: OffsetClamp, want: 0, ok: true},
		{name: "above-range-clamp", col: 5, mode: OffsetClamp, want: 11, ok: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RuneOffsetFromGraphemeColInLine(line, tc.col, tc.mode)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("off=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestLineConversionHelpers_InvalidClampMode(t *testing.T) {
	if _, ok := GraphemeColFromRuneOffsetInLine("a", 0, OffsetClampMode(99)); ok {
		t.Fatalf("expected invalid clamp mode to fail")
	}
	if _, ok := RuneOffsetFromGraphemeColInLine("a", 0, OffsetClampMode(99)); ok {
		t.Fatalf("expected invalid clamp mode to fail")
	}
}
