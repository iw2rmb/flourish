package main

import (
	"os"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor editor.Model
}

func newModel() model {
	cfg := editor.Config{
		Text: strings.Join([]string{
			"Virtual text example",
			"TODO lines get a view-only overlay annotation.",
			"API key values are visually redacted using deletions+insertions.",
			"",
			"TODO: add retry logic",
			"API_KEY=\"abcd1234\"",
		}, "\n"),
		ShowLineNums: true,
		Style:        editor.DefaultStyle(),
		VirtualTextProvider: func(ctx editor.VirtualTextContext) editor.VirtualText {
			var vt editor.VirtualText

			if strings.Contains(ctx.LineText, "TODO:") {
				vt.Insertions = append(vt.Insertions, editor.VirtualInsertion{
					GraphemeCol: utf8.RuneCountInString(ctx.LineText),
					Text:        "  <- pending",
					Role:        editor.VirtualRoleOverlay,
				})
			}

			if strings.Contains(ctx.LineText, "API_KEY=") {
				firstQuote := strings.Index(ctx.LineText, "\"")
				lastQuote := strings.LastIndex(ctx.LineText, "\"")
				if firstQuote >= 0 && lastQuote > firstQuote {
					startCol := utf8.RuneCountInString(ctx.LineText[:firstQuote+1])
					endCol := utf8.RuneCountInString(ctx.LineText[:lastQuote])
					vt.Deletions = append(vt.Deletions, editor.VirtualDeletion{
						StartGraphemeCol: startCol,
						EndGraphemeCol:   endCol,
					})
					vt.Insertions = append(vt.Insertions, editor.VirtualInsertion{
						GraphemeCol: startCol,
						Text:        "********",
						Role:        editor.VirtualRoleOverlay,
					})
				}
			}

			return vt
		},
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
