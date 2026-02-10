package editor

import "testing"

func TestWrapSegments_Grapheme_CoversLineAndRespectsWidth(t *testing.T) {
	vl := BuildVisualLine("abcdef", VirtualText{}, 4)
	segs := wrapSegmentsForVisualLine(vl, WrapGrapheme, 2)
	if len(segs) != 3 {
		t.Fatalf("segment count: got %d, want %d", len(segs), 3)
	}

	want := []wrappedSegment{
		{StartCol: 0, EndCol: 2, Cells: 2},
		{StartCol: 2, EndCol: 4, Cells: 2},
		{StartCol: 4, EndCol: 6, Cells: 2},
	}
	for i := range want {
		got := segs[i]
		if got.StartCol != want[i].StartCol || got.EndCol != want[i].EndCol || got.Cells != want[i].Cells {
			t.Fatalf("segment %d: got %+v, want %+v", i, got, want[i])
		}
	}
}

func TestWrapSegments_Word_WhitespaceAndLongTokenFallback(t *testing.T) {
	word := BuildVisualLine("hello world", VirtualText{}, 4)
	wordSegs := wrapSegmentsForVisualLine(word, WrapWord, 6)
	if len(wordSegs) != 2 {
		t.Fatalf("word segment count: got %d, want %d", len(wordSegs), 2)
	}
	if got, want := wordSegs[0], (wrappedSegment{StartCol: 0, EndCol: 6, Cells: 6}); got.StartCol != want.StartCol || got.EndCol != want.EndCol || got.Cells != want.Cells {
		t.Fatalf("word segment[0]: got %+v, want %+v", got, want)
	}
	if got, want := wordSegs[1], (wrappedSegment{StartCol: 6, EndCol: 11, Cells: 5}); got.StartCol != want.StartCol || got.EndCol != want.EndCol || got.Cells != want.Cells {
		t.Fatalf("word segment[1]: got %+v, want %+v", got, want)
	}

	longToken := BuildVisualLine("abcdefghij", VirtualText{}, 4)
	fallbackSegs := wrapSegmentsForVisualLine(longToken, WrapWord, 4)
	if len(fallbackSegs) != 3 {
		t.Fatalf("fallback segment count: got %d, want %d", len(fallbackSegs), 3)
	}
	for i, seg := range fallbackSegs {
		if seg.Cells > 4 {
			t.Fatalf("fallback segment %d exceeds width: got %d, max %d", i, seg.Cells, 4)
		}
	}
	if got, want := fallbackSegs[0].StartCol, 0; got != want {
		t.Fatalf("fallback start col: got %d, want %d", got, want)
	}
	if got, want := fallbackSegs[len(fallbackSegs)-1].EndCol, 10; got != want {
		t.Fatalf("fallback final end col: got %d, want %d", got, want)
	}
}

func TestWrapSegments_Word_PunctuationHeuristic(t *testing.T) {
	vl := BuildVisualLine("abc,def", VirtualText{}, 4)
	segs := wrapSegmentsForVisualLine(vl, WrapWord, 3)
	if len(segs) < 2 {
		t.Fatalf("segment count: got %d, want >=2", len(segs))
	}

	lineRunes := []rune("abc,def")
	for i := 1; i < len(segs); i++ {
		start := segs[i].StartCol
		if start >= 0 && start < len(lineRunes) && lineRunes[start] == ',' {
			t.Fatalf("segment %d starts with punctuation at col %d", i, start)
		}
	}
}

func TestWrapSegments_EmptyAndSpaceOnlyStable(t *testing.T) {
	empty := BuildVisualLine("", VirtualText{}, 4)
	emptySegs := wrapSegmentsForVisualLine(empty, WrapWord, 4)
	if len(emptySegs) != 1 {
		t.Fatalf("empty segment count: got %d, want %d", len(emptySegs), 1)
	}
	if got := emptySegs[0]; got.StartCol != 0 || got.EndCol != 0 || got.Cells != 0 {
		t.Fatalf("empty segment: got %+v, want {StartCol:0 EndCol:0 Cells:0}", got)
	}

	spaces := BuildVisualLine("     ", VirtualText{}, 4)
	spaceSegs := wrapSegmentsForVisualLine(spaces, WrapWord, 2)
	if len(spaceSegs) == 0 {
		t.Fatalf("space-only line produced no segments")
	}
	if got, want := spaceSegs[0].startCell, 0; got != want {
		t.Fatalf("space first segment startCell: got %d, want %d", got, want)
	}
	if got, want := spaceSegs[len(spaceSegs)-1].endCell, spaces.VisualLen(); got != want {
		t.Fatalf("space final endCell: got %d, want %d", got, want)
	}
	for i, seg := range spaceSegs {
		if seg.Cells > 2 {
			t.Fatalf("space segment %d exceeds width: got %d, max %d", i, seg.Cells, 2)
		}
	}
}
