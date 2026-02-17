package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor   editor.Model
	wrapMode editor.WrapMode
	width    int
	height   int
}

func newModel() model {
	cfg := editor.Config{
		Text: strings.Join([]string{
			"Wrap modes example",
			"Use ctrl+n (none), ctrl+w (word), ctrl+g (grapheme).",
			"Then resize your terminal and move the cursor through long lines.",
			"",
			"A very_long_token_without_spaces_that_forces_grapheme_fallback_in_word_mode.",
			"This sentence has natural word boundaries and demonstrates word wrapping behavior.",
		}, "\n"),
		ShowLineNums: true,
		Style:        editor.DefaultStyle(),
		WrapMode:     editor.WrapNone,
	}
	return model{editor: editor.New(cfg), wrapMode: editor.WrapNone}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor = m.editor.SetSize(msg.Width, editorHeight(msg.Height))
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			return m, tea.Quit
		case "ctrl+n":
			m = m.setWrapMode(editor.WrapNone)
			return m, nil
		case "ctrl+w":
			m = m.setWrapMode(editor.WrapWord)
			return m, nil
		case "ctrl+g":
			m = m.setWrapMode(editor.WrapGrapheme)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m model) View() string {
	status := fmt.Sprintf("Mode: %s | ctrl+n none, ctrl+w word, ctrl+g grapheme | ctrl+q quit", wrapLabel(m.wrapMode))
	return status + "\n" + m.editor.View()
}

func (m model) setWrapMode(mode editor.WrapMode) model {
	if m.wrapMode == mode {
		return m
	}

	buf := m.editor.Buffer()
	text := buf.Text()
	cursor := buf.Cursor()
	selRaw, hasSel := buf.SelectionRaw()
	focused := m.editor.Focused()

	next := editor.New(editor.Config{
		Text:         text,
		ShowLineNums: true,
		Style:        editor.DefaultStyle(),
		WrapMode:     mode,
	})
	next = next.SetSize(m.width, editorHeight(m.height))
	if !focused {
		next = next.Blur()
	}
	nextBuf := next.Buffer()
	nextBuf.SetCursor(cursor)
	if hasSel {
		nextBuf.SetSelection(selRaw)
	}

	m.editor = next
	m.wrapMode = mode
	return m
}

func editorHeight(total int) int {
	if total <= 1 {
		return 0
	}
	return total - 1
}

func wrapLabel(mode editor.WrapMode) string {
	switch mode {
	case editor.WrapNone:
		return "WrapNone"
	case editor.WrapWord:
		return "WrapWord"
	case editor.WrapGrapheme:
		return "WrapGrapheme"
	default:
		return "Unknown"
	}
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
