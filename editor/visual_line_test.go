package editor

import (
	"fmt"
	"testing"
)

func TestVisualLine_Mapping_DeletionRemovesColumns(t *testing.T) {
	vl := BuildVisualLine("abcd", VirtualText{
		Deletions: []VirtualDeletion{{StartGraphemeCol: 1, EndGraphemeCol: 3}}, // hide "bc"
	}, 4)

	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocGraphemeCol), "[0 3]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
	if got := vl.DocGraphemeColToVisualCell; len(got) != 5 {
		t.Fatalf("doc->visual len: got %d, want %d", len(got), 5)
	}
	if got, want := vl.DocGraphemeColToVisualCell[1], 1; got != want {
		t.Fatalf("doc col 1 maps to: got %d, want %d", got, want)
	}
	if got, want := vl.DocGraphemeColToVisualCell[2], 1; got != want {
		t.Fatalf("doc col 2 maps to: got %d, want %d", got, want)
	}
}

func TestVisualLine_Mapping_InsertionAddsCellsButDocStaysAnchored(t *testing.T) {
	vl := BuildVisualLine("ab", VirtualText{
		Insertions: []VirtualInsertion{{GraphemeCol: 1, Text: "XX"}},
	}, 4)

	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocGraphemeCol), "[0 1 1 1]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
	if got, want := vl.DocGraphemeColToVisualCell[1], 3; got != want {
		t.Fatalf("doc col 1 visual cell: got %d, want %d", got, want)
	}
}

func TestVisualLine_Mapping_WideGraphemeMapsAllCellsToOneDocCol(t *testing.T) {
	vl := BuildVisualLine("界", VirtualText{}, 4)
	if got, want := len(vl.VisualCellToDocGraphemeCol), 2; got != want {
		t.Fatalf("visual len: got %d, want %d", got, want)
	}
	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocGraphemeCol), "[0 0]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
}

func TestVisualLine_Mapping_TabExpansionDeterministic(t *testing.T) {
	vl := BuildVisualLine("a\tb", VirtualText{}, 4)
	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocGraphemeCol), "[0 1 1 1 2]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
}

func TestVisualLine_Mapping_CombiningClusterIsSingleGrapheme(t *testing.T) {
	vl := BuildVisualLine("e\u0301x", VirtualText{
		Deletions: []VirtualDeletion{{StartGraphemeCol: 0, EndGraphemeCol: 1}},
	}, 4)

	if got, want := vl.RawGraphemeLen, 2; got != want {
		t.Fatalf("raw grapheme len: got %d, want %d", got, want)
	}
	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocGraphemeCol), "[1]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
	if got, want := vl.DocGraphemeColToVisualCell[0], 0; got != want {
		t.Fatalf("deleted grapheme maps to next visible cell: got %d, want %d", got, want)
	}
}
