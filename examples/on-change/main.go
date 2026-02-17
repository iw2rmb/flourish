package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
	"github.com/iw2rmb/flourish/editor"
)

type eventState struct {
	count int
	last  editor.ChangeEvent
}

func (s *eventState) handleChange(ev editor.ChangeEvent) {
	s.count++
	s.last = ev
}

func (s *eventState) initFromBuffer(buf *buffer.Buffer) {
	s.last.Version = buf.Version()
	s.last.Cursor = buf.Cursor()
	s.last.Text = buf.Text()
	if r, ok := buf.Selection(); ok {
		s.last.Selection.Active = true
		s.last.Selection.Range = r
	}
}

type model struct {
	editor editor.Model
	events *eventState
}

func newModel() model {
	state := &eventState{}
	cfg := editor.Config{
		Text: strings.Join([]string{
			"OnChange example",
			"Edit this text. Status below updates on effective mutations.",
			"Movement and selection updates also fire events when state changes.",
			"Ctrl+Q quits.",
		}, "\n"),
		ShowLineNums: true,
		Style:        editor.DefaultStyle(),
		OnChange:     state.handleChange,
	}

	m := model{editor: editor.New(cfg), events: state}
	m.events.initFromBuffer(m.editor.Buffer())
	return m
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
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m model) View() string {
	selection := "none"
	if m.events.last.Selection.Active {
		r := m.events.last.Selection.Range
		selection = fmt.Sprintf("[%d:%d -> %d:%d)", r.Start.Row, r.Start.GraphemeCol, r.End.Row, r.End.GraphemeCol)
	}

	status := strings.Join([]string{
		"",
		"OnChange status:",
		fmt.Sprintf("events: %d", m.events.count),
		fmt.Sprintf("version: %d", m.events.last.Version),
		fmt.Sprintf("cursor: row=%d col=%d", m.events.last.Cursor.Row, m.events.last.Cursor.GraphemeCol),
		fmt.Sprintf("selection: %s", selection),
		fmt.Sprintf("text bytes: %d", len(m.events.last.Text)),
	}, "\n")

	return m.editor.View() + status
}

func editorHeight(total int) int {
	h := total - 7
	if h < 0 {
		return 0
	}
	return h
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
