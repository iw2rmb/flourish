package editor

import (
	"io"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/muesli/termenv"

	"github.com/iw2rmb/flourish/buffer"
)

func TestLinkProvider_RendersHyperlinksWithDefaultLinkStyle(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	textStyle := r.NewStyle()
	linkStyle := r.NewStyle().Underline(true)

	m := New(Config{
		Text: "abcd",
		Style: Style{
			Text:   textStyle,
			Cursor: r.NewStyle().Reverse(true),
			Link:   linkStyle,
		},
		LinkProvider: func(ctx LinkContext) ([]LinkSpan, error) {
			return []LinkSpan{{StartGraphemeCol: 1, EndGraphemeCol: 3, Target: "https://example.com"}}, nil
		},
	})
	m = m.SetSize(10, 1)
	m = m.Blur()

	got := m.renderContent()
	want := textStyle.Render("a") +
		renderHyperlink("https://example.com", linkStyle.Inherit(textStyle).Render("b")) +
		renderHyperlink("https://example.com", linkStyle.Inherit(textStyle).Render("c")) +
		textStyle.Render("d")
	if got != want {
		t.Fatalf("unexpected hyperlink render:\n got: %q\nwant: %q", got, want)
	}
}

func TestLinkAt_DocAndScreenCoordinates(t *testing.T) {
	m := New(Config{
		Text: "**a**",
		VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
			return VirtualText{
				Deletions: []VirtualDeletion{
					{StartGraphemeCol: 0, EndGraphemeCol: 2},
					{StartGraphemeCol: 3, EndGraphemeCol: 5},
				},
			}
		},
		LinkProvider: func(ctx LinkContext) ([]LinkSpan, error) {
			return []LinkSpan{{StartGraphemeCol: 2, EndGraphemeCol: 3, Target: "https://example.com/a"}}, nil
		},
	})
	m = m.SetSize(4, 1)

	hit, ok := m.LinkAt(buffer.Pos{Row: 0, GraphemeCol: 2})
	if !ok {
		t.Fatalf("LinkAt must resolve link at raw col 2")
	}
	if hit != (LinkHit{Row: 0, StartGraphemeCol: 2, EndGraphemeCol: 3, Target: "https://example.com/a"}) {
		t.Fatalf("unexpected LinkAt hit: got %+v", hit)
	}

	if _, ok := m.LinkAt(buffer.Pos{Row: 0, GraphemeCol: 1}); ok {
		t.Fatalf("LinkAt must not resolve outside link range")
	}

	screenHit, ok := m.LinkAtScreen(0, 0)
	if !ok {
		t.Fatalf("LinkAtScreen must resolve visible link cell")
	}
	if screenHit != hit {
		t.Fatalf("LinkAtScreen mismatch: got %+v, want %+v", screenHit, hit)
	}
}

func TestLinkProvider_ErrorFallsBackToPlainText(t *testing.T) {
	st := Style{Text: lipgloss.NewStyle(), Cursor: lipgloss.NewStyle().Reverse(true)}
	m := New(Config{
		Text:  "abcd",
		Style: st,
		LinkProvider: func(ctx LinkContext) ([]LinkSpan, error) {
			return []LinkSpan{{StartGraphemeCol: 1, EndGraphemeCol: 3, Target: "https://example.com"}}, io.EOF
		},
	})
	m = m.SetSize(10, 1)
	m = m.Blur()

	got := m.renderContent()
	want := st.Text.Render("a") + st.Text.Render("b") + st.Text.Render("c") + st.Text.Render("d")
	if got != want {
		t.Fatalf("unexpected render with link provider error:\n got: %q\nwant: %q", got, want)
	}
}
