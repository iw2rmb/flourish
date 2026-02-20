package editor

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

func completionRowBaseStyle(st Style, selected bool) lipgloss.Style {
	if selected {
		return st.CompletionSelected
	}
	return st.CompletionItem
}

func resolveCompletionSegmentStyle(
	base lipgloss.Style,
	styleForKey func(string) (lipgloss.Style, bool),
	item CompletionItem,
	seg CompletionSegment,
) lipgloss.Style {
	if styleForKey != nil && seg.StyleKey != "" {
		if keyed, ok := styleForKey(seg.StyleKey); ok {
			return keyed.Inherit(base)
		}
	}
	if styleForKey != nil && item.StyleKey != "" {
		if keyed, ok := styleForKey(item.StyleKey); ok {
			return keyed.Inherit(base)
		}
	}
	return base
}

func truncateCompletionSegments(segments []CompletionSegment, width int) []CompletionSegment {
	if width <= 0 || len(segments) == 0 {
		return nil
	}

	used := 0
	out := make([]CompletionSegment, 0, len(segments))
	appendText := func(styleKey, text string) {
		if text == "" {
			return
		}
		if len(out) > 0 && out[len(out)-1].StyleKey == styleKey {
			out[len(out)-1].Text += text
			return
		}
		out = append(out, CompletionSegment{Text: text, StyleKey: styleKey})
	}

	for _, seg := range segments {
		text := sanitizeCompletionSegmentText(seg.Text)
		if text == "" {
			continue
		}
		for _, gr := range graphemeutil.Split(text) {
			if used >= width {
				break
			}
			w := graphemeCellWidth(gr, used, 4)
			if w < 1 {
				w = 1
			}
			if used+w > width {
				remaining := width - used
				appendText(seg.StyleKey, strings.Repeat(" ", remaining))
				used = width
				break
			}
			appendText(seg.StyleKey, gr)
			used += w
		}
		if used >= width {
			break
		}
	}

	return out
}

func sanitizeCompletionSegmentText(s string) string {
	s = sanitizeSingleLine(s)
	if s == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}
