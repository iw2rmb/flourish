package editor

import (
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iw2rmb/flourish/buffer"
)

func TestRender_GhostInsertionAtEOL_UsesGhostStyle(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:   r.NewStyle(),
		Cursor: r.NewStyle().Reverse(true),
		Ghost:  r.NewStyle().Faint(true),
	}

	m := New(Config{
		Text:  "ab",
		Style: st,
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 2}, End: buffer.Pos{Row: 0, GraphemeCol: 2}},
					Text:  "X",
				}},
			}, true
		},
	})

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL

	got := m.renderContent()
	want := st.Text.Render("a") + st.Text.Render("b") + st.Cursor.Render(" ") + st.Ghost.Inherit(st.Text).Render("X")
	if got != want {
		t.Fatalf("unexpected ghost rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestRender_GhostInsertionAtNonEOL_UsesGhostStyle(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:   r.NewStyle(),
		Cursor: r.NewStyle().Reverse(true),
		Ghost:  r.NewStyle().Faint(true),
	}

	m := New(Config{
		Text:  "ab",
		Style: st,
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{
						Start: buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
						End:   buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
					},
					Text: "X",
				}},
			}, true
		},
	})

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1}) // non-EOL

	got := m.renderContent()
	want := st.Text.Render("a") + st.Ghost.Inherit(st.Text).Render("X") + st.Cursor.Render("b")
	if got != want {
		t.Fatalf("unexpected ghost rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestUpdate_GhostAccept_TabAppliesEdits(t *testing.T) {
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 2}, End: buffer.Pos{Row: 0, GraphemeCol: 2}},
					Text:  "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := m.buf.Text(); got != "abX" {
		t.Fatalf("text after ghost accept: got %q, want %q", got, "abX")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 3}) {
		t.Fatalf("cursor after ghost accept: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 3})
	}
}

func TestUpdate_GhostAccept_RightAppliesEdits(t *testing.T) {
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{Start: buffer.Pos{Row: 0, GraphemeCol: 2}, End: buffer.Pos{Row: 0, GraphemeCol: 2}},
					Text:  "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.buf.Text(); got != "abX" {
		t.Fatalf("text after ghost accept: got %q, want %q", got, "abX")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 3}) {
		t.Fatalf("cursor after ghost accept: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 3})
	}
}

func TestUpdate_GhostAccept_TabAppliesEdits_NonEOL(t *testing.T) {
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{
						Start: buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
						End:   buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
					},
					Text: "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1}) // non-EOL

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := m.buf.Text(); got != "aXb" {
		t.Fatalf("text after non-EOL ghost accept: got %q, want %q", got, "aXb")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor after non-EOL ghost accept: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
}

func TestUpdate_GhostAccept_RightAppliesEdits_NonEOL(t *testing.T) {
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{
						Start: buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
						End:   buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
					},
					Text: "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1}) // non-EOL

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.buf.Text(); got != "aXb" {
		t.Fatalf("text after non-EOL ghost accept: got %q, want %q", got, "aXb")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
		t.Fatalf("cursor after non-EOL ghost accept: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
	}
}

func TestUpdate_GhostAccept_EmptyEditsFallbacks(t *testing.T) {
	t.Run("RightFallsThroughToMovement", func(t *testing.T) {
		m := New(Config{
			Text: "ab",
			GhostProvider: func(ctx GhostContext) (Ghost, bool) {
				return Ghost{Text: "X"}, true
			},
		})
		m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1}) // non-EOL

		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
		if got := m.buf.Text(); got != "ab" {
			t.Fatalf("text after Right fallback: got %q, want %q", got, "ab")
		}
		if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
			t.Fatalf("cursor after Right fallback: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
		}
	})

	t.Run("TabFallsThroughToTabInsert", func(t *testing.T) {
		m := New(Config{
			Text: "ab",
			GhostProvider: func(ctx GhostContext) (Ghost, bool) {
				return Ghost{Text: "X"}, true
			},
		})
		m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1}) // non-EOL

		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		if got := m.buf.Text(); got != "a\tb" {
			t.Fatalf("text after Tab fallback: got %q, want %q", got, "a\\tb")
		}
		if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, GraphemeCol: 2}) {
			t.Fatalf("cursor after Tab fallback: got %v, want %v", got, buffer.Pos{Row: 0, GraphemeCol: 2})
		}
	})
}

func TestRender_GhostSuppressedWhenCompletionVisible(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:   r.NewStyle(),
		Cursor: r.NewStyle().Reverse(true),
		Ghost:  r.NewStyle().Faint(true),
	}

	m := New(Config{
		Text:  "ab",
		Style: st,
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{
						Start: buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
						End:   buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
					},
					Text: "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1})
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Items: []CompletionItem{
			{ID: "0", InsertText: "x"},
		},
		VisibleIndices: []int{0},
	})

	got := m.renderContent()
	want := st.Text.Render("a") + st.Cursor.Render("b")
	if got != want {
		t.Fatalf("ghost should be suppressed while completion popup is visible:\n got: %q\nwant: %q", got, want)
	}
}

func TestUpdate_GhostAccept_RightSuppressedWhenCompletionVisible(t *testing.T) {
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{
						Start: buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
						End:   buffer.Pos{Row: 0, GraphemeCol: ctx.GraphemeCol},
					},
					Text: "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1})
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Items: []CompletionItem{
			{ID: "0", InsertText: "y"},
		},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("right key should not accept ghost while completion visible: got %q, want %q", got, want)
	}
	if got, want := m.buf.Cursor(), (buffer.Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("right key should fall through to cursor movement: got %v, want %v", got, want)
	}
	if !m.CompletionState().Visible {
		t.Fatalf("completion popup should remain visible")
	}
}
