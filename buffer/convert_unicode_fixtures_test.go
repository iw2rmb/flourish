package buffer

import "testing"

type conversionBoundary struct {
	pos      Pos
	byteOff  int
	runeOff  int
}

type conversionFixture struct {
	name             string
	text             string
	boundaries       []conversionBoundary
	invalidByteOffs  []int
	invalidRuneOffs  []int
}

func unicodeConversionFixtures() []conversionFixture {
	return []conversionFixture{
		{
			name: "ascii-single",
			text: "a",
			boundaries: []conversionBoundary{
				{pos: Pos{Row: 0, GraphemeCol: 0}, byteOff: 0, runeOff: 0},
				{pos: Pos{Row: 0, GraphemeCol: 1}, byteOff: 1, runeOff: 1},
			},
		},
		{
			name: "multibyte-utf8",
			text: "é",
			boundaries: []conversionBoundary{
				{pos: Pos{Row: 0, GraphemeCol: 0}, byteOff: 0, runeOff: 0},
				{pos: Pos{Row: 0, GraphemeCol: 1}, byteOff: 2, runeOff: 1},
			},
			invalidByteOffs: []int{1},
		},
		{
			name: "combining-mark",
			text: "e\u0301",
			boundaries: []conversionBoundary{
				{pos: Pos{Row: 0, GraphemeCol: 0}, byteOff: 0, runeOff: 0},
				{pos: Pos{Row: 0, GraphemeCol: 1}, byteOff: 3, runeOff: 2},
			},
			invalidByteOffs: []int{1, 2},
			invalidRuneOffs: []int{1},
		},
		{
			name: "zwj-emoji",
			text: "👨‍👩‍👧‍👦",
			boundaries: []conversionBoundary{
				{pos: Pos{Row: 0, GraphemeCol: 0}, byteOff: 0, runeOff: 0},
				{pos: Pos{Row: 0, GraphemeCol: 1}, byteOff: 25, runeOff: 7},
			},
			invalidByteOffs: []int{1, 4, 10, 24},
			invalidRuneOffs: []int{1, 3, 6},
		},
		{
			name: "multiline-boundaries",
			text: "a\nb",
			boundaries: []conversionBoundary{
				{pos: Pos{Row: 0, GraphemeCol: 0}, byteOff: 0, runeOff: 0},
				{pos: Pos{Row: 0, GraphemeCol: 1}, byteOff: 1, runeOff: 1},
				{pos: Pos{Row: 1, GraphemeCol: 0}, byteOff: 2, runeOff: 2},
				{pos: Pos{Row: 1, GraphemeCol: 1}, byteOff: 3, runeOff: 3},
			},
		},
	}
}

func strictConvertPolicy() ConvertPolicy {
	return ConvertPolicy{ClampMode: OffsetError, NewlineMode: NewlineAsSingleRune}
}

func clampConvertPolicy() ConvertPolicy {
	return ConvertPolicy{ClampMode: OffsetClamp, NewlineMode: NewlineAsSingleRune}
}

func TestBuffer_UnicodeFixtures_BoundaryRoundTrip(t *testing.T) {
	p := strictConvertPolicy()

	for _, fx := range unicodeConversionFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			b := New(fx.text, Options{})

			for _, boundary := range fx.boundaries {
				gotPos, ok := b.PosFromByteOffset(boundary.byteOff, p)
				if !ok || gotPos != boundary.pos {
					t.Fatalf("PosFromByteOffset(%d)=(%v,%v), want (%v,true)", boundary.byteOff, gotPos, ok, boundary.pos)
				}

				gotPos, ok = b.PosFromRuneOffset(boundary.runeOff, p)
				if !ok || gotPos != boundary.pos {
					t.Fatalf("PosFromRuneOffset(%d)=(%v,%v), want (%v,true)", boundary.runeOff, gotPos, ok, boundary.pos)
				}

				gotByte, ok := b.ByteOffsetFromPos(boundary.pos, p)
				if !ok || gotByte != boundary.byteOff {
					t.Fatalf("ByteOffsetFromPos(%v)=(%d,%v), want (%d,true)", boundary.pos, gotByte, ok, boundary.byteOff)
				}

				gotRune, ok := b.RuneOffsetFromPos(boundary.pos, p)
				if !ok || gotRune != boundary.runeOff {
					t.Fatalf("RuneOffsetFromPos(%v)=(%d,%v), want (%d,true)", boundary.pos, gotRune, ok, boundary.runeOff)
				}
			}
		})
	}
}

func TestBuffer_UnicodeFixtures_RejectInteriorOffsets(t *testing.T) {
	p := strictConvertPolicy()

	for _, fx := range unicodeConversionFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			b := New(fx.text, Options{})

			for _, off := range fx.invalidByteOffs {
				if _, ok := b.PosFromByteOffset(off, p); ok {
					t.Fatalf("PosFromByteOffset(%d) should fail for interior byte offset", off)
				}
			}
			for _, off := range fx.invalidRuneOffs {
				if _, ok := b.PosFromRuneOffset(off, p); ok {
					t.Fatalf("PosFromRuneOffset(%d) should fail for interior rune offset", off)
				}
			}
		})
	}
}

func TestBuffer_UnicodeFixtures_BoundsPolicies(t *testing.T) {
	strict := strictConvertPolicy()
	clamp := clampConvertPolicy()

	for _, fx := range unicodeConversionFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			b := New(fx.text, Options{})
			start := fx.boundaries[0].pos
			end := fx.boundaries[len(fx.boundaries)-1].pos
			maxByte := fx.boundaries[len(fx.boundaries)-1].byteOff
			maxRune := fx.boundaries[len(fx.boundaries)-1].runeOff

			if _, ok := b.PosFromByteOffset(-1, strict); ok {
				t.Fatalf("PosFromByteOffset(-1) should fail in error mode")
			}
			if _, ok := b.PosFromByteOffset(maxByte+1, strict); ok {
				t.Fatalf("PosFromByteOffset(max+1) should fail in error mode")
			}
			if _, ok := b.PosFromRuneOffset(-1, strict); ok {
				t.Fatalf("PosFromRuneOffset(-1) should fail in error mode")
			}
			if _, ok := b.PosFromRuneOffset(maxRune+1, strict); ok {
				t.Fatalf("PosFromRuneOffset(max+1) should fail in error mode")
			}

			if got, ok := b.PosFromByteOffset(-1, clamp); !ok || got != start {
				t.Fatalf("PosFromByteOffset(-1)=(%v,%v), want (%v,true)", got, ok, start)
			}
			if got, ok := b.PosFromByteOffset(maxByte+1, clamp); !ok || got != end {
				t.Fatalf("PosFromByteOffset(max+1)=(%v,%v), want (%v,true)", got, ok, end)
			}
			if got, ok := b.PosFromRuneOffset(-1, clamp); !ok || got != start {
				t.Fatalf("PosFromRuneOffset(-1)=(%v,%v), want (%v,true)", got, ok, start)
			}
			if got, ok := b.PosFromRuneOffset(maxRune+1, clamp); !ok || got != end {
				t.Fatalf("PosFromRuneOffset(max+1)=(%v,%v), want (%v,true)", got, ok, end)
			}
		})
	}
}
