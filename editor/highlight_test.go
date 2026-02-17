package editor

import (
	"errors"
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iw2rmb/flourish/buffer"
)

func TestHighlighting_CalledOnlyForVisibleLines(t *testing.T) {
	var rows []int
	h := &stubHighlighter{
		fn: func(ctx LineContext) ([]HighlightSpan, error) {
			rows = append(rows, ctx.Row)
			return nil, nil
		},
	}

	m := New(Config{
		Text:        "a\nb\nc",
		Highlighter: h,
	})
	rows = nil

	m = m.SetSize(10, 1)
	rows = nil
	_ = m.renderContent()

	if len(rows) != 1 || rows[0] != 0 {
		t.Fatalf("highlighter rows: got %v, want %v", rows, []int{0})
	}
}

func TestHighlighting_ErrorFallsBackToPlainText(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{Text: r.NewStyle()}

	m := New(Config{
		Text:  "abcd",
		Style: st,
		Highlighter: &stubHighlighter{
			fn: func(ctx LineContext) ([]HighlightSpan, error) {
				return []HighlightSpan{{StartGraphemeCol: 1, EndGraphemeCol: 3, Style: r.NewStyle().Underline(true)}}, errors.New("boom")
			},
		},
	})
	m = m.SetSize(10, 1)
	m = m.Blur()

	got := m.renderContent()
	want := st.Text.Render("a") + st.Text.Render("b") + st.Text.Render("c") + st.Text.Render("d")
	if got != want {
		t.Fatalf("unexpected render with highlighter error:\n got: %q\nwant: %q", got, want)
	}
}

func TestHighlighting_RebuildsAfterAutoFollowScroll(t *testing.T) {
	var rows []int
	h := &stubHighlighter{
		fn: func(ctx LineContext) ([]HighlightSpan, error) {
			rows = append(rows, ctx.Row)
			return nil, nil
		},
	}

	m := New(Config{
		Text:        "a\nb\nc",
		Highlighter: h,
	})
	m = m.SetSize(10, 1)
	rows = nil

	m.buf.SetCursor(buffer.Pos{Row: 2, GraphemeCol: 0})
	m, _ = m.Update(struct{}{})

	if len(rows) != 2 || rows[0] != 0 || rows[1] != 2 {
		t.Fatalf("highlighter rows after auto-follow scroll: got %v, want %v", rows, []int{0, 2})
	}
}

func TestHighlighting_AppliesSpansToVisibleDocText(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	textStyle := r.NewStyle()
	hlStyle := r.NewStyle().Underline(true)
	st := Style{Text: textStyle}

	m := New(Config{
		Text:  "abcd",
		Style: st,
		Highlighter: &stubHighlighter{
			fn: func(ctx LineContext) ([]HighlightSpan, error) {
				return []HighlightSpan{{StartGraphemeCol: 1, EndGraphemeCol: 3, Style: hlStyle}}, nil
			},
		},
	})
	m = m.SetSize(10, 1)
	m = m.Blur()

	got := m.renderContent()
	want := textStyle.Render("a") + hlStyle.Inherit(textStyle).Render("b") + hlStyle.Inherit(textStyle).Render("c") + textStyle.Render("d")
	if got != want {
		t.Fatalf("unexpected highlighted render:\n got: %q\nwant: %q", got, want)
	}
}

type stubHighlighter struct {
	fn func(LineContext) ([]HighlightSpan, error)
}

func (s *stubHighlighter) HighlightLine(ctx LineContext) ([]HighlightSpan, error) {
	if s.fn == nil {
		return nil, nil
	}
	return s.fn(ctx)
}
