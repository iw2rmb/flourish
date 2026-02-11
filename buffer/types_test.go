package buffer

import "testing"

func TestComparePos(t *testing.T) {
	t.Run("row", func(t *testing.T) {
		if got := ComparePos(Pos{Row: 0, GraphemeCol: 0}, Pos{Row: 1, GraphemeCol: 0}); got >= 0 {
			t.Fatalf("expected < 0, got %d", got)
		}
		if got := ComparePos(Pos{Row: 2, GraphemeCol: 0}, Pos{Row: 1, GraphemeCol: 999}); got <= 0 {
			t.Fatalf("expected > 0, got %d", got)
		}
	})

	t.Run("col", func(t *testing.T) {
		if got := ComparePos(Pos{Row: 1, GraphemeCol: 0}, Pos{Row: 1, GraphemeCol: 1}); got >= 0 {
			t.Fatalf("expected < 0, got %d", got)
		}
		if got := ComparePos(Pos{Row: 1, GraphemeCol: 2}, Pos{Row: 1, GraphemeCol: 1}); got <= 0 {
			t.Fatalf("expected > 0, got %d", got)
		}
	})

	t.Run("equal", func(t *testing.T) {
		if got := ComparePos(Pos{Row: 3, GraphemeCol: 4}, Pos{Row: 3, GraphemeCol: 4}); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})
}

func TestNormalizeRange(t *testing.T) {
	r := NormalizeRange(Range{Start: Pos{Row: 2, GraphemeCol: 3}, End: Pos{Row: 1, GraphemeCol: 9}})
	if r.Start != (Pos{Row: 1, GraphemeCol: 9}) || r.End != (Pos{Row: 2, GraphemeCol: 3}) {
		t.Fatalf("unexpected range: %#v", r)
	}

	r2 := NormalizeRange(r)
	if r2 != r {
		t.Fatalf("expected idempotent normalize: %#v != %#v", r2, r)
	}
}

func TestClampPos(t *testing.T) {
	lineLens := []int{1, 0, 3}
	ll := func(row int) int { return lineLens[row] }

	cases := []struct {
		in   Pos
		want Pos
	}{
		{in: Pos{Row: -1, GraphemeCol: -1}, want: Pos{Row: 0, GraphemeCol: 0}},
		{in: Pos{Row: 999, GraphemeCol: 999}, want: Pos{Row: 2, GraphemeCol: 3}},
		{in: Pos{Row: 1, GraphemeCol: 5}, want: Pos{Row: 1, GraphemeCol: 0}},
		{in: Pos{Row: 0, GraphemeCol: 1}, want: Pos{Row: 0, GraphemeCol: 1}},
	}

	for _, tc := range cases {
		if got := ClampPos(tc.in, len(lineLens), ll); got != tc.want {
			t.Fatalf("ClampPos(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
