package editor

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iw2rmb/flouris/buffer"
)

func TestRender_LineNumberAlignment_1To120(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 120; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("x")
	}

	m := New(Config{
		Text:         sb.String(),
		ShowLineNums: true,
	})
	m = m.Blur()
	m = m.SetSize(10, 120)

	lines := strings.Split(m.View(), "\n")
	if len(lines) != 120 {
		t.Fatalf("expected 120 lines, got %d", len(lines))
	}

	digits := 3
	for i, line := range lines {
		wantPrefix := fmt.Sprintf("%*d ", digits, i+1)
		gotLine := stripANSI(line)
		if !strings.HasPrefix(gotLine, wantPrefix) {
			t.Fatalf("line %d prefix: got %q, want prefix %q", i+1, gotLine, wantPrefix)
		}
	}
}

func TestRender_CursorStyleAppliedWhenFocused(t *testing.T) {
	m := New(Config{
		Text:  "ab",
		Style: Style{Text: lipgloss.NewStyle(), Cursor: lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)},
	})

	got := m.renderContent()
	want := " a b"
	if got != want {
		t.Fatalf("unexpected cursor rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestRender_Selection_MultiLine_HalfOpen(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:      r.NewStyle(),
		Selection: r.NewStyle().Underline(true),
		Cursor:    r.NewStyle().Reverse(true),
	}

	m := New(Config{Text: "ab\ncd\nef", Style: st})
	m = m.Blur()

	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 1},
		End:   buffer.Pos{Row: 2, GraphemeCol: 1},
	})

	got := m.renderContent()
	want := strings.Join([]string{
		st.Text.Render("a") + st.Selection.Render("b"),
		st.Selection.Render("cd"),
		st.Selection.Render("e") + st.Text.Render("f"),
	}, "\n")

	if got != want {
		t.Fatalf("unexpected selection rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestRender_Selection_LineBoundary_EndAtStartOfNextLine(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:      r.NewStyle(),
		Selection: r.NewStyle().Underline(true),
	}

	m := New(Config{Text: "ab\ncd", Style: st})
	m = m.Blur()

	// Half-open: selecting [0:1, 1:0) selects only "b".
	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 1},
		End:   buffer.Pos{Row: 1, GraphemeCol: 0},
	})

	got := m.renderContent()
	want := strings.Join([]string{
		st.Text.Render("a") + st.Selection.Render("b"),
		st.Text.Render("cd"),
	}, "\n")

	if got != want {
		t.Fatalf("unexpected line-boundary selection rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestRender_Selection_EmptySelectionRendersAsText(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:      r.NewStyle(),
		Selection: r.NewStyle().Underline(true),
	}

	m := New(Config{Text: "ab", Style: st})
	m = m.Blur()

	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 1},
		End:   buffer.Pos{Row: 0, GraphemeCol: 1},
	})

	got := m.renderContent()
	want := st.Text.Render("ab")
	if got != want {
		t.Fatalf("unexpected empty selection rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestRender_Selection_FullLineSelection(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:      r.NewStyle(),
		Selection: r.NewStyle().Underline(true),
	}

	m := New(Config{Text: "ab\ncd", Style: st})
	m = m.Blur()

	// Half-open: selecting [0:0, 1:0) selects the full first line only.
	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 0},
		End:   buffer.Pos{Row: 1, GraphemeCol: 0},
	})

	got := m.renderContent()
	want := strings.Join([]string{
		st.Selection.Render("ab"),
		st.Text.Render("cd"),
	}, "\n")

	if got != want {
		t.Fatalf("unexpected full-line selection rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestRender_Selection_DeletedRangesDoNotRenderHighlight(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)

	st := Style{
		Text:      r.NewStyle(),
		Selection: r.NewStyle().Underline(true),
	}

	m := New(Config{
		Text:  "**a**",
		Style: st,
		VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
			// Hide markdown markers: "**a**" -> "a"
			return VirtualText{
				Deletions: []VirtualDeletion{
					{StartGraphemeCol: 0, EndGraphemeCol: 2},
					{StartGraphemeCol: 3, EndGraphemeCol: 5},
				},
			}
		},
	})
	m = m.Blur()

	// Select the entire raw line; only the visible doc-backed cell should be highlighted.
	m.buf.SetSelection(buffer.Range{
		Start: buffer.Pos{Row: 0, GraphemeCol: 0},
		End:   buffer.Pos{Row: 0, GraphemeCol: 5},
	})

	got := m.renderContent()
	want := st.Selection.Render("a")
	if got != want {
		t.Fatalf("unexpected selection rendering with deletions:\n got: %q\nwant: %q", got, want)
	}
}

func TestRender_HorizontalScroll_ClipsByXOffset_TabAndWideGrapheme(t *testing.T) {
	st := Style{
		Text:   lipgloss.NewStyle(),
		Cursor: lipgloss.NewStyle().Reverse(true), // ensure New() doesn't replace Style with defaults
	}

	m := New(Config{
		Text:     "a\tb",
		TabWidth: 4,
		Style:    st,
	})
	m = m.Blur()
	m = m.SetSize(3, 1)

	m.xOffset = 0
	if got := stripANSI(m.renderContent()); got != "a  " {
		t.Fatalf("tab clip xOffset=0: got %q, want %q", got, "a  ")
	}

	m.xOffset = 2
	if got := stripANSI(m.renderContent()); got != "  b" {
		t.Fatalf("tab clip xOffset=2: got %q, want %q", got, "  b")
	}

	m2 := New(Config{
		Text:  "a界b",
		Style: st,
	})
	m2 = m2.Blur()
	m2 = m2.SetSize(2, 1)

	// "界" is 2 cells wide; starting the slice at its second cell renders a blank for alignment.
	m2.xOffset = 2
	if got := stripANSI(m2.renderContent()); got != " b" {
		t.Fatalf("wide grapheme clip xOffset=2: got %q, want %q", got, " b")
	}
}

func TestRender_SoftWrap_GraphemeAndLineNumbers(t *testing.T) {
	st := Style{
		Text:   lipgloss.NewStyle(),
		Cursor: lipgloss.NewStyle().Reverse(true),
	}

	m := New(Config{
		Text:     "abcdef",
		WrapMode: WrapGrapheme,
		Style:    st,
	})
	m = m.Blur()
	m = m.SetSize(3, 2)

	if got := stripANSI(m.renderContent()); got != "abc\ndef" {
		t.Fatalf("wrapped content: got %q, want %q", got, "abc\ndef")
	}

	withNums := New(Config{
		Text:         "abcdef",
		WrapMode:     WrapGrapheme,
		ShowLineNums: true,
		Style:        st,
	})
	withNums = withNums.Blur()
	withNums = withNums.SetSize(5, 2) // gutter 2 + content 3

	if got := stripANSI(withNums.renderContent()); got != "1 abc\n  def" {
		t.Fatalf("wrapped content with line nums: got %q, want %q", got, "1 abc\n  def")
	}
}
