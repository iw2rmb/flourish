package editor

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func BenchmarkCursorMove(b *testing.B) {
	for _, lines := range []int{100, 1000, 3000} {
		doc := benchmarkCursorDoc(lines)

		b.Run(fmt.Sprintf("plain/lines=%d", lines), func(b *testing.B) {
			m := New(Config{Text: doc})
			m = m.SetSize(160, 40)
			benchmarkCursorPingPong(b, m)
		})

		b.Run(fmt.Sprintf("virtual_text/lines=%d", lines), func(b *testing.B) {
			m := New(Config{
				Text: doc,
				VirtualTextProvider: func(ctx VirtualTextContext) VirtualText {
					if !ctx.HasCursor {
						return VirtualText{}
					}
					return VirtualText{
						Insertions: []VirtualInsertion{{
							GraphemeCol: ctx.CursorGraphemeCol,
							Text:        ">",
							Role:        VirtualRoleOverlay,
						}},
					}
				},
			})
			m = m.SetSize(160, 40)
			benchmarkCursorPingPong(b, m)
		})
	}
}

func benchmarkCursorPingPong(b *testing.B, m Model) {
	right := tea.KeyMsg{Type: tea.KeyRight}
	left := tea.KeyMsg{Type: tea.KeyLeft}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			m, _ = m.Update(right)
		} else {
			m, _ = m.Update(left)
		}
	}
}

func benchmarkCursorDoc(lines int) string {
	if lines <= 0 {
		return ""
	}
	const line = "- [Bubble Tea](https://github.com/charmbracelet/bubbletea) and `code` token"
	return strings.TrimSuffix(strings.Repeat(line+"\n", lines), "\n")
}
