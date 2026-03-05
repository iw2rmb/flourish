package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/iw2rmb/flourish/buffer"
	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor      editor.Model
	store       *rowMarkStore
	lastVersion uint64
}

type rowMarkStore struct {
	marks map[int]editor.RowMarkState
}

func newModel() model {
	store := &rowMarkStore{marks: map[int]editor.RowMarkState{}}

	st := editor.DefaultStyle()
	st.RowMarkInserted = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	st.RowMarkUpdated = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	st.RowMarkDeleted = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)

	cfg := editor.Config{
		Text: strings.Join([]string{
			"Row marks example",
			"Edit normally to produce marks from local changes.",
			"Host controls colors/symbols and mark state via provider.",
			"",
			"Inserted rows get I, updated rows get U.",
			"Deleted-row anchors use v (above) or ^ (below).",
			"Ctrl+Q quits.",
		}, "\n"),
		Gutter:   editor.LineNumberGutter(),
		Style:    st,
		WrapMode: editor.WrapWord,
		Scrollbar: editor.ScrollbarConfig{
			Horizontal: editor.ScrollbarNever,
		},
		RowMarkProvider: func(ctx editor.RowMarkContext) editor.RowMarkState {
			return store.marks[ctx.Row]
		},
		RowMarkWidth: 1,
		RowMarkSymbols: editor.RowMarkSymbols{
			Inserted:     "I",
			Updated:      "U",
			DeletedAbove: "v",
			DeletedBelow: "^",
		},
	}

	ed := editor.New(cfg)
	return model{
		editor:      ed,
		store:       store,
		lastVersion: ed.Buffer().Version(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.editor = m.editor.SetSize(msg.Width, editorHeight(msg.Height))
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	m.processLatestChange()
	return m, cmd
}

func (m model) View() tea.View {
	cursorRow := m.editor.Buffer().Cursor().Row
	current := m.store.marks[cursorRow]
	status := fmt.Sprintf(
		"normal editing updates marks | ctrl+q quit | row=%d mark=%+v",
		cursorRow+1,
		current,
	)
	return tea.NewView(m.editor.View().Content + "\n" + status)
}

func (m *model) processLatestChange() {
	buf := m.editor.Buffer()
	if buf == nil {
		return
	}
	v := buf.Version()
	if v == m.lastVersion {
		return
	}
	ch, ok := buf.LastChange()
	if !ok {
		m.lastVersion = v
		return
	}
	if ch.VersionAfter <= m.lastVersion || len(ch.AppliedEdits) == 0 {
		m.lastVersion = ch.VersionAfter
		return
	}

	lineCount := buf.LineCount()
	for _, edit := range ch.AppliedEdits {
		m.applyEditMark(edit, lineCount)
	}
	m.pruneMarks(lineCount)
	m.lastVersion = ch.VersionAfter
	m.editor = m.editor.InvalidateGutter()
}

func (m *model) applyEditMark(edit buffer.AppliedEdit, lineCount int) {
	before := edit.RangeBefore
	after := edit.RangeAfter

	beforeSpan := before.End.Row - before.Start.Row
	afterSpan := after.End.Row - after.Start.Row
	deltaRows := afterSpan - beforeSpan

	m.remapMarks(before.Start.Row, before.End.Row, deltaRows)

	// Always treat the edited start row as updated.
	m.setMark(after.Start.Row, lineCount, editor.RowMarkState{Updated: true})

	switch {
	case deltaRows > 0:
		// Mark net-new rows as inserted.
		for row := after.Start.Row + 1; row <= after.Start.Row+deltaRows; row++ {
			m.setMark(row, lineCount, editor.RowMarkState{Inserted: true})
		}
	case deltaRows < 0:
		// Anchor deleted rows at the surviving boundary row.
		anchor := clampRow(after.Start.Row, lineCount)
		if anchor >= 0 {
			m.setMark(anchor, lineCount, editor.RowMarkState{DeletedAbove: true})
		}
	default:
		end := clampRow(after.End.Row, lineCount)
		start := clampRow(after.Start.Row, lineCount)
		if start >= 0 && end >= 0 && end >= start {
			for row := start; row <= end; row++ {
				m.setMark(row, lineCount, editor.RowMarkState{Updated: true})
			}
		}
	}
}

func (m *model) remapMarks(startRow, endRow, deltaRows int) {
	if len(m.store.marks) == 0 {
		return
	}

	next := make(map[int]editor.RowMarkState, len(m.store.marks))
	for row, state := range m.store.marks {
		switch {
		case row < startRow:
			next[row] = state
		case row > endRow:
			next[row+deltaRows] = state
		default:
			// Drop marks for touched rows and let this change re-apply fresh marks.
		}
	}
	m.store.marks = next
}

func (m *model) setMark(row, lineCount int, add editor.RowMarkState) {
	row = clampRow(row, lineCount)
	if row < 0 {
		return
	}
	cur := m.store.marks[row]
	cur.Inserted = cur.Inserted || add.Inserted
	cur.Updated = cur.Updated || add.Updated
	cur.DeletedAbove = cur.DeletedAbove || add.DeletedAbove
	cur.DeletedBelow = cur.DeletedBelow || add.DeletedBelow
	m.store.marks[row] = cur
}

func (m *model) pruneMarks(lineCount int) {
	for row, st := range m.store.marks {
		if row < 0 || row >= lineCount {
			delete(m.store.marks, row)
			continue
		}
		if !st.Inserted && !st.Updated && !st.DeletedAbove && !st.DeletedBelow {
			delete(m.store.marks, row)
		}
	}
}

func clampRow(row, lineCount int) int {
	if lineCount <= 0 {
		return -1
	}
	if row < 0 {
		return 0
	}
	if row >= lineCount {
		return lineCount - 1
	}
	return row
}

func editorHeight(total int) int {
	h := total - 1
	if h < 0 {
		return 0
	}
	return h
}

func main() {
	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
