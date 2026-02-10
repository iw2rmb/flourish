package editor

import "github.com/charmbracelet/lipgloss"

// Style controls the editor's rendering.
//
// This is a minimal subset used in Phase 5.
type Style struct {
	Gutter        lipgloss.Style
	LineNum       lipgloss.Style
	LineNumActive lipgloss.Style

	Text   lipgloss.Style
	Cursor lipgloss.Style
}

func DefaultStyle() Style {
	gutter := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return Style{
		Gutter:        gutter,
		LineNum:       gutter,
		LineNumActive: lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true),
		Text:          lipgloss.NewStyle(),
		Cursor:        lipgloss.NewStyle().Reverse(true),
	}
}
