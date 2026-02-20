package editor

import (
	"strings"
	"testing"

	"github.com/iw2rmb/flourish/buffer"
)

func TestCompletionPopupRender_BelowAnchor(t *testing.T) {
	m := New(Config{Text: "000000\n111111\n222222"})
	m = m.Blur()
	m = m.SetSize(6, 3)
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  bufferPos(0, 1),
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "abc"}}},
			{ID: "1", Label: []CompletionSegment{{Text: "xyz"}}},
		},
	})

	got := strings.Split(stripANSI(m.View()), "\n")
	want := []string{"000000", "1abc11", "2xyz22"}
	assertLines(t, got, want)
}

func TestCompletionPopupRender_FlipsAboveWhenNoSpaceBelow(t *testing.T) {
	m := New(Config{Text: "000000\n111111\n222222"})
	m = m.Blur()
	m = m.SetSize(6, 3)
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  bufferPos(2, 2),
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "ab"}}},
			{ID: "1", Label: []CompletionSegment{{Text: "cd"}}},
		},
	})

	got := strings.Split(stripANSI(m.View()), "\n")
	want := []string{"00ab00", "11cd11", "222222"}
	assertLines(t, got, want)
}

func TestCompletionPopupRender_HidesWhenAnchorIsOffscreen(t *testing.T) {
	m := New(Config{Text: "000000\n111111\n222222\n333333"})
	m = m.Blur()
	m = m.SetSize(6, 2)
	m.viewport.YOffset = 2

	base := stripANSI(m.View())

	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  bufferPos(0, 0),
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "abc"}}},
		},
	})

	if got := stripANSI(m.View()); got != base {
		t.Fatalf("offscreen anchor popup should not render:\n got: %q\nwant: %q", got, base)
	}
}

func TestCompletionPopupRender_ClampWidthAndRows(t *testing.T) {
	m := New(Config{
		Text:                     "000000\n111111\n222222\n333333",
		CompletionMaxVisibleRows: 2,
		CompletionMaxWidth:       4,
	})
	m = m.Blur()
	m = m.SetSize(6, 4)
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  bufferPos(0, 5),
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "abcdef"}}},
			{ID: "1", Label: []CompletionSegment{{Text: "ghijkl"}}},
			{ID: "2", Label: []CompletionSegment{{Text: "mnopqr"}}},
		},
	})

	got := strings.Split(stripANSI(m.View()), "\n")
	want := []string{"000000", "11abcd", "22ghij", "333333"}
	assertLines(t, got, want)
}

func TestCompletionPopupRender_DefaultLimitsClampToSmallViewport(t *testing.T) {
	m := New(Config{
		Text:                     "000\n111",
		CompletionMaxVisibleRows: -1,
		CompletionMaxWidth:       -1,
	})
	m = m.Blur()
	m = m.SetSize(3, 2)
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  bufferPos(0, 0),
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "abcdef"}}},
			{ID: "1", Label: []CompletionSegment{{Text: "gh"}}},
		},
	})

	got := strings.Split(stripANSI(m.View()), "\n")
	want := []string{"000", "abc"}
	assertLines(t, got, want)
}

func TestCompletionPopupRender_WrapPlacementUsesProjectedAnchor(t *testing.T) {
	m := New(Config{
		Text:     "abcdef\nghijkl",
		WrapMode: WrapGrapheme,
	})
	m = m.Blur()
	m = m.SetSize(3, 4)
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  bufferPos(0, 4),
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "XY"}}},
		},
	})

	got := strings.Split(stripANSI(m.View()), "\n")
	want := []string{"abc", "def", "gXY", "jkl"}
	assertLines(t, got, want)
}

func TestCompletionPopupRender_WrapNonePlacementRespectsXOffset(t *testing.T) {
	m := New(Config{Text: "abcdefghi\n123456789"})
	m = m.Blur()
	m = m.SetSize(5, 2)
	m.xOffset = 4
	m.rebuildContent()

	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  bufferPos(0, 6),
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "zz"}}},
		},
	})

	got := strings.Split(stripANSI(m.View()), "\n")
	want := []string{"efghi", "56zz9"}
	assertLines(t, got, want)
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("line count: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func bufferPos(row, col int) buffer.Pos {
	return buffer.Pos{Row: row, GraphemeCol: col}
}
