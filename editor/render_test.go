package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
		if !strings.HasPrefix(line, wantPrefix) {
			t.Fatalf("line %d prefix: got %q, want prefix %q", i+1, line, wantPrefix)
		}
	}
}

func TestRender_CursorProducesANSIWhenFocused(t *testing.T) {
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
