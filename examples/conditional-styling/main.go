package main

import (
	"os"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor editor.Model
}

type tokenHighlighter struct {
	words map[string]lipgloss.Style
}

func (h tokenHighlighter) HighlightLine(ctx editor.LineContext) ([]editor.HighlightSpan, error) {
	var spans []editor.HighlightSpan
	for word, style := range h.words {
		spans = append(spans, wordSpans(ctx.Text, word, style)...)
	}
	return spans, nil
}

func newModel() model {
	highlightBase := lipgloss.NewStyle().Foreground(lipgloss.Color("111"))

	cfg := editor.Config{
		Text: strings.Join([]string{
			"# sshd_config excerpt",
			"PasswordAuthentication yes",
			"PubkeyAuthentication yes",
			"AuthorizedKeysFile .ssh/authorized_keys",
			"PermitRootLogin no",
			"",
			"Use arrow keys to move active row, Ctrl+Q to quit.",
		}, "\n"),
		Gutter: editor.LineNumberGutter(),
		Style:  editor.DefaultStyle(),
		Highlighter: tokenHighlighter{
			words: map[string]lipgloss.Style{
				"PasswordAuthentication": highlightBase,
				"PubkeyAuthentication":   highlightBase,
				"AuthorizedKeysFile":     highlightBase,
				"yes":                    highlightBase,
				"no":                     highlightBase,
			},
		},
		RowStyleForRow: func(ctx editor.RowStyleContext) (lipgloss.Style, bool) {
			if ctx.IsActive {
				return lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					BorderStyle(lipgloss.ThickBorder()).
					BorderLeft(true), true
			}
			return lipgloss.Style{}, false
		},
		TokenStyleForToken: func(ctx editor.TokenStyleContext) (lipgloss.Style, bool) {
			if !ctx.IsHighlighted {
				return lipgloss.Style{}, false
			}
			if ctx.IsActiveRow {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true), true
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("117")), true
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
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+q" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m model) View() tea.View { return m.editor.View() }

func wordSpans(line, word string, style lipgloss.Style) []editor.HighlightSpan {
	if line == "" || word == "" {
		return nil
	}

	var spans []editor.HighlightSpan
	for offset := 0; offset < len(line); {
		i := strings.Index(line[offset:], word)
		if i < 0 {
			break
		}
		startByte := offset + i
		endByte := startByte + len(word)
		if isWordBoundary(line, startByte, endByte) {
			startCol := utf8.RuneCountInString(line[:startByte])
			endCol := startCol + utf8.RuneCountInString(word)
			spans = append(spans, editor.HighlightSpan{
				StartGraphemeCol: startCol,
				EndGraphemeCol:   endCol,
				Style:            style,
			})
		}
		offset = endByte
	}
	return spans
}

func isWordBoundary(line string, startByte, endByte int) bool {
	leftOK := startByte == 0 || !isWordByte(line[startByte-1])
	rightOK := endByte == len(line) || !isWordByte(line[endByte])
	return leftOK && rightOK
}

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func main() {
	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
