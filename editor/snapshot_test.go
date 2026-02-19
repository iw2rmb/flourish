package editor

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

type snapshotNoopHighlighter struct{}

func (snapshotNoopHighlighter) HighlightLine(LineContext) ([]HighlightSpan, error) {
	return nil, nil
}

func TestRenderSnapshot_NonEmptyAndStableForSameFrame(t *testing.T) {
	m := New(Config{
		Text:         "ab\ncd\nef",
		ShowLineNums: true,
	})
	m = m.SetSize(8, 2)

	s1 := m.RenderSnapshot()
	if s1.Token == 0 {
		t.Fatalf("snapshot token must be non-zero")
	}
	if got, want := s1.Viewport.VisibleRows, 2; got != want {
		t.Fatalf("snapshot visible rows: got %d, want %d", got, want)
	}
	if len(s1.Rows) == 0 {
		t.Fatalf("snapshot rows must be non-empty")
	}
	if got := s1.Rows[0].SegmentIndex; got != 0 {
		t.Fatalf("first row segment index: got %d, want %d", got, 0)
	}

	s2 := m.RenderSnapshot()
	if s2.Token != s1.Token {
		t.Fatalf("same frame token mismatch: got %d, want %d", s2.Token, s1.Token)
	}
	if !reflect.DeepEqual(s2, s1) {
		t.Fatalf("same frame snapshot content mismatch")
	}

	if len(s1.Rows[0].VisibleDocCols) > 0 {
		s1.Rows[0].VisibleDocCols[0] = 999
	}
	s3 := m.RenderSnapshot()
	if len(s3.Rows[0].VisibleDocCols) > 0 && s3.Rows[0].VisibleDocCols[0] == 999 {
		t.Fatalf("snapshot clone isolation violated")
	}
}

func TestRenderSnapshot_SegmentIndexForWrappedRows(t *testing.T) {
	m := New(Config{
		Text:     "abcdef\ngh",
		WrapMode: WrapGrapheme,
	})
	m = m.SetSize(3, 3)

	s := m.RenderSnapshot()
	if len(s.Rows) < 3 {
		t.Fatalf("wrapped snapshot rows: got %d, want at least %d", len(s.Rows), 3)
	}

	if got, want := s.Rows[0].DocRow, 0; got != want {
		t.Fatalf("row0 doc row: got %d, want %d", got, want)
	}
	if got, want := s.Rows[0].SegmentIndex, 0; got != want {
		t.Fatalf("row0 segment index: got %d, want %d", got, want)
	}
	if got, want := s.Rows[1].DocRow, 0; got != want {
		t.Fatalf("row1 doc row: got %d, want %d", got, want)
	}
	if got, want := s.Rows[1].SegmentIndex, 1; got != want {
		t.Fatalf("row1 segment index: got %d, want %d", got, want)
	}
	if got, want := s.Rows[2].DocRow, 1; got != want {
		t.Fatalf("row2 doc row: got %d, want %d", got, want)
	}
	if got, want := s.Rows[2].SegmentIndex, 0; got != want {
		t.Fatalf("row2 segment index: got %d, want %d", got, want)
	}
}

func TestRenderSnapshot_TokenInvalidationMatrix(t *testing.T) {
	t.Run("buffer version change", func(t *testing.T) {
		m := New(Config{Text: "ab"})
		m = m.SetSize(8, 1)
		t0 := m.RenderSnapshot().Token

		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
		t1 := m.RenderSnapshot().Token
		if t1 == t0 {
			t.Fatalf("token must change after buffer mutation")
		}
	})

	t.Run("viewport yoffset change", func(t *testing.T) {
		m := New(Config{Text: "0\n1\n2\n3"})
		m = m.SetSize(8, 2)
		t0 := m.RenderSnapshot().Token

		m, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
		t1 := m.RenderSnapshot().Token
		if t1 == t0 {
			t.Fatalf("token must change after viewport yoffset change")
		}
	})

	t.Run("horizontal offset change", func(t *testing.T) {
		m := New(Config{Text: "abcdef"})
		m = m.SetSize(3, 1)
		t0 := m.RenderSnapshot().Token

		m.xOffset = 2
		t1 := m.RenderSnapshot().Token
		if t1 == t0 {
			t.Fatalf("token must change after horizontal offset change")
		}
	})

	t.Run("wrap mode change", func(t *testing.T) {
		m := New(Config{Text: "abcdef"})
		m = m.SetSize(3, 1)
		t0 := m.RenderSnapshot().Token

		m.cfg.WrapMode = WrapGrapheme
		t1 := m.RenderSnapshot().Token
		if t1 == t0 {
			t.Fatalf("token must change after wrap mode change")
		}
	})

	t.Run("focus change", func(t *testing.T) {
		m := New(Config{Text: "ab"})
		m = m.SetSize(3, 1)
		t0 := m.RenderSnapshot().Token

		m = m.Blur()
		t1 := m.RenderSnapshot().Token
		if t1 == t0 {
			t.Fatalf("token must change after focus change")
		}
	})

	t.Run("highlighter change", func(t *testing.T) {
		m := New(Config{Text: "ab"})
		m = m.SetSize(3, 1)
		t0 := m.RenderSnapshot().Token

		h := snapshotNoopHighlighter{}
		m.cfg.Highlighter = h
		t1 := m.RenderSnapshot().Token
		if t1 == t0 {
			t.Fatalf("token must change after highlighter-affecting change")
		}
	})

	t.Run("no changes keeps token", func(t *testing.T) {
		m := New(Config{Text: "ab"})
		m = m.SetSize(8, 1)
		t0 := m.RenderSnapshot().Token
		t1 := m.RenderSnapshot().Token
		if t1 != t0 {
			t.Fatalf("token must stay stable when state is unchanged")
		}
	})
}

func TestSnapshotMapping_RejectsStaleAndMatchesFresh(t *testing.T) {
	m := New(Config{Text: "ab\ncd\nef"})
	m = m.SetSize(10, 2)

	s0 := m.RenderSnapshot()
	wantPos := m.ScreenToDoc(1, 0)
	gotPos, ok := m.ScreenToDocWithSnapshot(s0, 1, 0)
	if !ok || gotPos != wantPos {
		t.Fatalf("fresh ScreenToDocWithSnapshot: got (%v,%v), want (%v,true)", gotPos, ok, wantPos)
	}

	wantX, wantY, wantOK := m.DocToScreen(buffer.Pos{Row: 0, GraphemeCol: 1})
	gotX, gotY, gotOK := m.DocToScreenWithSnapshot(s0, buffer.Pos{Row: 0, GraphemeCol: 1})
	if gotX != wantX || gotY != wantY || gotOK != wantOK {
		t.Fatalf("fresh DocToScreenWithSnapshot mismatch: got (%d,%d,%v), want (%d,%d,%v)", gotX, gotY, gotOK, wantX, wantY, wantOK)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if _, ok := m.ScreenToDocWithSnapshot(s0, 1, 0); ok {
		t.Fatalf("stale snapshot must be rejected for ScreenToDocWithSnapshot")
	}
	if _, _, ok := m.DocToScreenWithSnapshot(s0, buffer.Pos{Row: 0, GraphemeCol: 1}); ok {
		t.Fatalf("stale snapshot must be rejected for DocToScreenWithSnapshot")
	}

	s1 := m.RenderSnapshot()
	if _, ok := m.ScreenToDocWithSnapshot(s1, 1, 0); !ok {
		t.Fatalf("fresh snapshot must be accepted")
	}
}

func TestSnapshotMapping_ParityWrapAndDecorated(t *testing.T) {
	m := New(Config{
		Text:         "abcdef\nghijkl",
		ShowLineNums: true,
		WrapMode:     WrapGrapheme,
		VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
			if ctx.Row == 0 {
				return VirtualText{
					Insertions: []VirtualInsertion{{GraphemeCol: 2, Text: "XX"}},
				}
			}
			return VirtualText{}
		},
	})
	m = m.SetSize(6, 2)
	s := m.RenderSnapshot()

	screenPoints := [][2]int{
		{0, 0},
		{2, 0},
		{4, 1},
		{99, 1},
	}
	for _, p := range screenPoints {
		want := m.ScreenToDoc(p[0], p[1])
		got, ok := m.ScreenToDocWithSnapshot(s, p[0], p[1])
		if !ok {
			t.Fatalf("ScreenToDocWithSnapshot returned ok=false for point (%d,%d)", p[0], p[1])
		}
		if got != want {
			t.Fatalf("ScreenToDoc parity mismatch at (%d,%d): got %v, want %v", p[0], p[1], got, want)
		}
	}

	docPoints := []buffer.Pos{
		{Row: 0, GraphemeCol: 1},
		{Row: 0, GraphemeCol: 5},
		{Row: 1, GraphemeCol: 2},
		{Row: 1, GraphemeCol: 6},
	}
	for _, pos := range docPoints {
		wantX, wantY, wantOK := m.DocToScreen(pos)
		gotX, gotY, gotOK := m.DocToScreenWithSnapshot(s, pos)
		if gotX != wantX || gotY != wantY || gotOK != wantOK {
			t.Fatalf("DocToScreen parity mismatch for %v: got (%d,%d,%v), want (%d,%d,%v)", pos, gotX, gotY, gotOK, wantX, wantY, wantOK)
		}
	}
}

func TestSnapshotMapping_ParityWrapNoneHorizontalOffsetAndOffscreen(t *testing.T) {
	m := New(Config{Text: "abcdef"})
	m = m.SetSize(3, 1)
	m.xOffset = 2
	s := m.RenderSnapshot()

	screenPoints := [][2]int{{0, 0}, {1, 0}, {2, 0}}
	for _, p := range screenPoints {
		want := m.ScreenToDoc(p[0], p[1])
		got, ok := m.ScreenToDocWithSnapshot(s, p[0], p[1])
		if !ok {
			t.Fatalf("ScreenToDocWithSnapshot returned ok=false for point (%d,%d)", p[0], p[1])
		}
		if got != want {
			t.Fatalf("ScreenToDoc parity mismatch at (%d,%d): got %v, want %v", p[0], p[1], got, want)
		}
	}

	docPoints := []buffer.Pos{
		{Row: 0, GraphemeCol: 5},
		{Row: 0, GraphemeCol: 0},
	}
	for _, pos := range docPoints {
		wantX, wantY, wantOK := m.DocToScreen(pos)
		gotX, gotY, gotOK := m.DocToScreenWithSnapshot(s, pos)
		if gotX != wantX || gotY != wantY || gotOK != wantOK {
			t.Fatalf("DocToScreen parity mismatch for %v: got (%d,%d,%v), want (%d,%d,%v)", pos, gotX, gotY, gotOK, wantX, wantY, wantOK)
		}
	}
}
