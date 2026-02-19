package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/editor"
)

type eventState struct {
	count   int
	hasLast bool
	last    editor.ChangeEvent
}

func (s *eventState) handleChange(ev editor.ChangeEvent) {
	s.count++
	s.hasLast = true
	s.last = ev
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
		Gutter:   editor.LineNumberGutter(),
		Style:    editor.DefaultStyle(),
		OnChange: state.handleChange,
	}

	m := model{editor: editor.New(cfg), events: state}
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
	statusLines := []string{
		"",
		"OnChange status:",
		fmt.Sprintf("events: %d", m.events.count),
	}
	if m.events.hasLast {
		ch := m.events.last.Change
		selection := "none"
		if ch.SelectionAfter.Active {
			r := ch.SelectionAfter.Range
			selection = fmt.Sprintf("[%d:%d -> %d:%d)", r.Start.Row, r.Start.GraphemeCol, r.End.Row, r.End.GraphemeCol)
		}

		editSummary := "none"
		if len(ch.AppliedEdits) > 0 {
			e := ch.AppliedEdits[len(ch.AppliedEdits)-1]
			editSummary = fmt.Sprintf(
				"insert=%q delete=%q before=[%d:%d -> %d:%d) after=[%d:%d -> %d:%d)",
				e.InsertText,
				e.DeletedText,
				e.RangeBefore.Start.Row, e.RangeBefore.Start.GraphemeCol,
				e.RangeBefore.End.Row, e.RangeBefore.End.GraphemeCol,
				e.RangeAfter.Start.Row, e.RangeAfter.Start.GraphemeCol,
				e.RangeAfter.End.Row, e.RangeAfter.End.GraphemeCol,
			)
		}

		statusLines = append(
			statusLines,
			fmt.Sprintf("version: %d -> %d", ch.VersionBefore, ch.VersionAfter),
			fmt.Sprintf(
				"cursor: row=%d col=%d -> row=%d col=%d",
				ch.CursorBefore.Row, ch.CursorBefore.GraphemeCol,
				ch.CursorAfter.Row, ch.CursorAfter.GraphemeCol,
			),
			fmt.Sprintf("selection after: %s", selection),
			fmt.Sprintf("applied edits: %d", len(ch.AppliedEdits)),
			fmt.Sprintf("last edit: %s", editSummary),
		)
	} else {
		statusLines = append(statusLines, "last change: none yet")
	}

	status := strings.Join(statusLines, "\n")

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
