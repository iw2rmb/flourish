package editor

import (
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iw2rmb/flouris/buffer"
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
					Range: buffer.Range{Start: buffer.Pos{Row: 0, Col: 2}, End: buffer.Pos{Row: 0, Col: 2}},
					Text:  "X",
				}},
			}, true
		},
	})

	m.buf.SetCursor(buffer.Pos{Row: 0, Col: 2}) // EOL

	got := m.renderContent()
	want := st.Text.Render("a") + st.Text.Render("b") + st.Cursor.Render(" ") + st.Ghost.Inherit(st.Text).Render("X")
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
					Range: buffer.Range{Start: buffer.Pos{Row: 0, Col: 2}, End: buffer.Pos{Row: 0, Col: 2}},
					Text:  "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, Col: 2}) // EOL

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := m.buf.Text(); got != "abX" {
		t.Fatalf("text after ghost accept: got %q, want %q", got, "abX")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, Col: 3}) {
		t.Fatalf("cursor after ghost accept: got %v, want %v", got, buffer.Pos{Row: 0, Col: 3})
	}
}

func TestUpdate_GhostAccept_RightAppliesEdits(t *testing.T) {
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			return Ghost{
				Text: "X",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{Start: buffer.Pos{Row: 0, Col: 2}, End: buffer.Pos{Row: 0, Col: 2}},
					Text:  "X",
				}},
			}, true
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, Col: 2}) // EOL

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.buf.Text(); got != "abX" {
		t.Fatalf("text after ghost accept: got %q, want %q", got, "abX")
	}
	if got := m.buf.Cursor(); got != (buffer.Pos{Row: 0, Col: 3}) {
		t.Fatalf("cursor after ghost accept: got %v, want %v", got, buffer.Pos{Row: 0, Col: 3})
	}
}
