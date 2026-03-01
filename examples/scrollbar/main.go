package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor editor.Model
}

const headerRows = 1

func newModel() model {
	st := editor.DefaultStyle()
	st.ScrollbarTrack = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	st.ScrollbarThumb = lipgloss.NewStyle().Background(lipgloss.Color("81"))
	st.ScrollbarCorner = lipgloss.NewStyle().Background(lipgloss.Color("236"))

	cfg := editor.Config{
		Text:         demoText(),
		Gutter:       editor.LineNumberGutter(),
		Style:        st,
		WrapMode:     editor.WrapNone,
		ScrollPolicy: editor.ScrollAllowManual,
		Scrollbar: editor.ScrollbarConfig{
			Vertical:   editor.ScrollbarAlways,
			Horizontal: editor.ScrollbarAlways,
			MinThumb:   2,
		},
	}

	return model{editor: editor.New(cfg)}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.editor = m.editor.SetSize(msg.Width, editorHeight(msg.Height))
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+q" {
			return m, tea.Quit
		}
	case tea.MouseMsg:
		msg = translateMouseToEditor(msg)
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m model) View() string {
	help := "Scrollbar demo: drag thumb, click track to page, wheel to scroll, ctrl+q quit."
	return help + "\n" + m.editor.View()
}

func editorHeight(total int) int {
	if total <= headerRows {
		return 0
	}
	return total - headerRows
}

func translateMouseToEditor(msg tea.MouseMsg) tea.MouseMsg {
	ev := tea.MouseEvent(msg)
	ev.Y -= headerRows
	return tea.MouseMsg(ev)
}

func demoText() string {
	lines := make([]string, 0, 130)
	lines = append(lines,
		"Scrollbar behavior demo.",
		"Use mouse on track/thumb for vertical and horizontal scrolling.",
		"",
	)

	longSegment := strings.Repeat("scrollbar-demo-", 14)
	for i := 1; i <= 120; i++ {
		lines = append(lines, fmt.Sprintf("row %03d | %s", i, longSegment))
	}
	return strings.Join(lines, "\n")
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
