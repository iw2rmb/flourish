package main

import (
	"os"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor editor.Model
}

type demoHighlighter struct {
	keywordStyle lipgloss.Style
	todoStyle    lipgloss.Style
}

func (h demoHighlighter) HighlightLine(ctx editor.LineContext) ([]editor.HighlightSpan, error) {
	var spans []editor.HighlightSpan
	spans = append(spans, wordSpans(ctx.Text, "func", h.keywordStyle)...)
	spans = append(spans, wordSpans(ctx.Text, "if", h.keywordStyle)...)
	spans = append(spans, wordSpans(ctx.Text, "return", h.keywordStyle)...)
	spans = append(spans, wordSpans(ctx.Text, "TODO", h.todoStyle)...)
	return spans, nil
}

func newModel() model {
	cfg := editor.Config{
		Text: strings.Join([]string{
			"Highlighter example",
			"Type TODO, func, if, or return to see styled spans.",
			"Ctrl+Q quits.",
			"",
			"func classify(v int) string {",
			"\tif v > 0 {",
			"\t\treturn \"positive\" // TODO: tune thresholds",
			"\t}",
			"\treturn \"zero\"",
			"}",
		}, "\n"),
		Gutter: editor.LineNumberGutter(),
		Style:  editor.DefaultStyle(),
		Highlighter: demoHighlighter{
			keywordStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
			todoStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
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
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
