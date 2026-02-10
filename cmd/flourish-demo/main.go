package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flouris/buffer"
	"github.com/iw2rmb/flouris/editor"
)

type model struct {
	editor editor.Model
}

func newModel() model {
	cfg := editor.Config{
		Text:         "Hello from flourish.\n\nType to edit.\nUse arrows to move.\nCtrl+C to quit.",
		ShowLineNums: true,
		Style:        editor.DefaultStyle(),
	}
	m := model{editor: editor.New(cfg)}
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.editor = m.editor.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		handleKey(m.editor.Buffer(), msg)
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m model) View() string { return m.editor.View() }

func handleKey(b *buffer.Buffer, msg tea.KeyMsg) {
	if b == nil {
		return
	}

	switch msg.String() {
	case "left":
		b.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirLeft})
	case "right":
		b.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirRight})
	case "up":
		b.Move(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirUp})
	case "down":
		b.Move(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirDown})
	case "home":
		b.Move(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirHome})
	case "end":
		b.Move(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirEnd})
	case "backspace":
		b.DeleteBackward()
	case "delete":
		b.DeleteForward()
	case "enter":
		b.InsertNewline()
	case "tab":
		b.InsertRune('\t')
	default:
		for _, r := range msg.Runes {
			b.InsertRune(r)
		}
	}
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
