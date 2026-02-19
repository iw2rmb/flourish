package main

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor editor.Model
}

func newModel() model {
	cfg := editor.Config{
		Text: strings.Join([]string{
			"Inline suggestions example",
			"Move to the fmt. line and press Tab or Right to accept the ghost suggestion.",
			"Ctrl+Q quits.",
			"",
			"package main",
			"",
			"import \"fmt\"",
			"",
			"func main() {",
			"\tfmt.",
			"}",
		}, "\n"),
		Gutter: editor.LineNumberGutter(),
		Style:  editor.DefaultStyle(),
		GhostProvider: func(ctx editor.GhostContext) (editor.Ghost, bool) {
			if !ctx.IsEndOfLine {
				return editor.Ghost{}, false
			}
			if strings.TrimSpace(ctx.LineText) != "fmt." {
				return editor.Ghost{}, false
			}

			suggestion := "Println(\"hello from ghost\")"
			pos := buffer.Pos{Row: ctx.Row, GraphemeCol: ctx.GraphemeCol}
			return editor.Ghost{
				Text: suggestion,
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{Start: pos, End: pos},
					Text:  suggestion,
				}},
			}, true
		},
		GhostAccept: editor.GhostAccept{AcceptTab: true, AcceptRight: true},
	}

	m := model{editor: editor.New(cfg)}
	m.editor.Buffer().SetCursor(buffer.Pos{Row: 9, GraphemeCol: 5})
	return m
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
