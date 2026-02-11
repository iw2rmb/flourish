package editor

import "testing"

func TestIterateGraphemeSteps_TabUsesTabStops(t *testing.T) {
	steps := iterateGraphemeSteps("a\tb", 4, 0)
	if len(steps) != 3 {
		t.Fatalf("step count: got %d, want %d", len(steps), 3)
	}

	if got, want := steps[0].CellWidth, 1; got != want {
		t.Fatalf("width of 'a': got %d, want %d", got, want)
	}
	if got, want := steps[1].CellWidth, 3; got != want {
		t.Fatalf("width of tab after col 1: got %d, want %d", got, want)
	}
	if got, want := steps[2].CellWidth, 1; got != want {
		t.Fatalf("width of 'b': got %d, want %d", got, want)
	}

	tabOnly := iterateGraphemeSteps("\t", 4, 2)
	if len(tabOnly) != 1 {
		t.Fatalf("tab-only step count: got %d, want %d", len(tabOnly), 1)
	}
	if got, want := tabOnly[0].CellWidth, 2; got != want {
		t.Fatalf("width of tab at visual col 2: got %d, want %d", got, want)
	}
}

func TestIterateGraphemeSteps_UnicodeBoundariesAndWidths(t *testing.T) {
	cases := []struct {
		name           string
		text           string
		wantFirstWidth int
	}{
		{name: "combining", text: "e\u0301x", wantFirstWidth: 1},
		{name: "emoji", text: "🙂x", wantFirstWidth: 2},
		{name: "cjk", text: "界x", wantFirstWidth: 2},
		{name: "zwj", text: "👨‍👩‍👧‍👦x", wantFirstWidth: 2},
		{name: "combining-only", text: "\u0301x", wantFirstWidth: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := iterateGraphemeSteps(tc.text, 4, 0)
			if len(steps) < 1 {
				t.Fatalf("no grapheme steps for %q", tc.text)
			}

			if got := steps[0].CellWidth; got != tc.wantFirstWidth {
				t.Fatalf("first width: got %d, want %d", got, tc.wantFirstWidth)
			}

			for i, st := range steps {
				if st.NextGraphemeCol <= st.GraphemeCol {
					t.Fatalf("step %d does not advance boundary: col=%d next=%d", i, st.GraphemeCol, st.NextGraphemeCol)
				}
				if st.CellWidth < 0 {
					t.Fatalf("step %d has negative width: %d", i, st.CellWidth)
				}
			}
		})
	}
}
