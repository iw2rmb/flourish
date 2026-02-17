package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor editor.Model
}

func newModel() model {
	cfg := editor.Config{
		Text:         "Simple example\n\nType to edit.\nUse arrows to move.\nCtrl+Q to quit.",
		ShowLineNums: true,
		Style:        editor.DefaultStyle(),
	}
	return model{editor: editor.New(cfg)}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.editor = m.editor.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+q" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m model) View() string { return m.editor.View() }

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
