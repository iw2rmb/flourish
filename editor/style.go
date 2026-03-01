package editor

import "charm.land/lipgloss/v2"

// Style controls the editor's rendering.
type Style struct {
	Gutter lipgloss.Style

	Text      lipgloss.Style
	Selection lipgloss.Style
	Cursor    lipgloss.Style
	Link      lipgloss.Style
	// Scrollbar styles are used for editor-owned scrollbar chrome.
	ScrollbarTrack  lipgloss.Style
	ScrollbarThumb  lipgloss.Style
	ScrollbarCorner lipgloss.Style

	CompletionItem     lipgloss.Style
	CompletionSelected lipgloss.Style

	Ghost          lipgloss.Style
	VirtualOverlay lipgloss.Style
}

// isZero returns true when every lipgloss.Style field is at its default
// (renders text unchanged). This replaces reflect.DeepEqual with short-circuit
// render checks.
func (s Style) isZero() bool {
	return isLipglossZero(s.Gutter) &&
		isLipglossZero(s.Text) &&
		isLipglossZero(s.Selection) &&
		isLipglossZero(s.Cursor) &&
		isLipglossZero(s.Link) &&
		isLipglossZero(s.ScrollbarTrack) &&
		isLipglossZero(s.ScrollbarThumb) &&
		isLipglossZero(s.ScrollbarCorner) &&
		isLipglossZero(s.CompletionItem) &&
		isLipglossZero(s.CompletionSelected) &&
		isLipglossZero(s.Ghost) &&
		isLipglossZero(s.VirtualOverlay)
}

func isLipglossZero(s lipgloss.Style) bool {
	return s.Render("x") == "x"
}

func DefaultStyle() Style {
	gutter := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return Style{
		Gutter:         gutter,
		Text:           lipgloss.NewStyle(),
		Selection:      lipgloss.NewStyle().Background(lipgloss.Color("237")),
		Cursor:         lipgloss.NewStyle().Reverse(true),
		Link:           lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Underline(true),
		ScrollbarTrack: lipgloss.NewStyle().Background(lipgloss.Color("236")),
		ScrollbarThumb: lipgloss.NewStyle().Background(lipgloss.Color("241")),
		ScrollbarCorner: lipgloss.NewStyle().
			Background(lipgloss.Color("236")),
		CompletionItem: lipgloss.NewStyle(),
		CompletionSelected: lipgloss.NewStyle().
			Background(lipgloss.Color("238")),
		Ghost: lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Faint(true),
		VirtualOverlay: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Faint(true),
	}
}
