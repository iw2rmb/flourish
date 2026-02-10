package editor

import (
	"fmt"
	"testing"
)

func TestVisualLine_Mapping_DeletionRemovesColumns(t *testing.T) {
	vl := BuildVisualLine("abcd", VirtualText{
		Deletions: []VirtualDeletion{{StartCol: 1, EndCol: 3}}, // hide "bc"
	}, 4)

	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocCol), "[0 3]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
	if got := vl.DocColToVisualCell; len(got) != 5 {
		t.Fatalf("doc->visual len: got %d, want %d", len(got), 5)
	}
	if got, want := vl.DocColToVisualCell[1], 1; got != want {
		t.Fatalf("doc col 1 maps to: got %d, want %d", got, want)
	}
	if got, want := vl.DocColToVisualCell[2], 1; got != want {
		t.Fatalf("doc col 2 maps to: got %d, want %d", got, want)
	}
}

func TestVisualLine_Mapping_InsertionAddsCellsButDocStaysAnchored(t *testing.T) {
	vl := BuildVisualLine("ab", VirtualText{
		Insertions: []VirtualInsertion{{Col: 1, Text: "XX"}},
	}, 4)

	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocCol), "[0 1 1 1]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
	if got, want := vl.DocColToVisualCell[1], 3; got != want {
		t.Fatalf("doc col 1 visual cell: got %d, want %d", got, want)
	}
}

func TestVisualLine_Mapping_WideGraphemeMapsAllCellsToOneDocCol(t *testing.T) {
	vl := BuildVisualLine("界", VirtualText{}, 4)
	if got, want := len(vl.VisualCellToDocCol), 2; got != want {
		t.Fatalf("visual len: got %d, want %d", got, want)
	}
	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocCol), "[0 0]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
}

func TestVisualLine_Mapping_TabExpansionDeterministic(t *testing.T) {
	vl := BuildVisualLine("a\tb", VirtualText{}, 4)
	if got, want := fmt.Sprintf("%v", vl.VisualCellToDocCol), "[0 1 1 1 2]"; got != want {
		t.Fatalf("visual->doc: got %s, want %s", got, want)
	}
}
