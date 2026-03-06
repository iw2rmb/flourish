package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/iw2rmb/flourish/editor"
)

type model struct {
	editor      editor.Model
	store       *rowMarkStore
	lastVersion uint64
	baseLines   []string
}

type rowMarkStore struct {
	marks map[int]editor.RowMarkState
}

func newModel() model {
	store := &rowMarkStore{marks: map[int]editor.RowMarkState{}}
	initialText := strings.Join([]string{
		"Row marks example",
		"Edit normally to produce marks from local changes.",
		"Host controls colors/symbols and mark state via provider.",
		"",
		"Inserted rows get I, updated rows get U.",
		"Deleted-row anchors use v (above) or ^ (below).",
		"Ctrl+Q quits.",
	}, "\n")

	st := editor.DefaultStyle()
	st.RowMarkInserted = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	st.RowMarkUpdated = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	st.RowMarkDeleted = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)

	cfg := editor.Config{
		Text:     initialText,
		Gutter:   editor.LineNumberGutter(),
		Style:    st,
		WrapMode: editor.WrapWord,
		Scrollbar: editor.ScrollbarConfig{
			Horizontal: editor.ScrollbarNever,
		},
		RowMarkProvider: func(ctx editor.RowMarkContext) editor.RowMarkState {
			return store.marks[ctx.Row]
		},
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
		baseLines:   strings.Split(initialText, "\n"),
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
		"marks are relative to initial snapshot | ctrl+q quit | row=%d mark=%+v",
		cursorRow+1,
		current,
	)
	v := m.editor.View()
	v.Content = v.Content + "\n" + status
	// Run this demo in alt-screen to avoid terminal scrollback/selection
	// shortcuts swallowing shifted arrow keys.
	v.AltScreen = true
	return v
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

	m.store.marks = computeRowMarks(m.baseLines, buf.RawLines())
	m.lastVersion = ch.VersionAfter
	m.editor = m.editor.InvalidateGutter()
}

func computeRowMarks(baseLines, curLines []string) map[int]editor.RowMarkState {
	n, m := len(baseLines), len(curLines)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if baseLines[i] == curLines[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
				continue
			}
			if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	type diffKind uint8
	const (
		diffEqual diffKind = iota
		diffDelete
		diffInsert
	)
	type diffOp struct{ kind diffKind }

	ops := make([]diffOp, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		if baseLines[i] == curLines[j] {
			ops = append(ops, diffOp{kind: diffEqual})
			i++
			j++
			continue
		}
		if lcs[i+1][j] >= lcs[i][j+1] {
			ops = append(ops, diffOp{kind: diffDelete})
			i++
			continue
		}
		ops = append(ops, diffOp{kind: diffInsert})
		j++
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{kind: diffDelete})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{kind: diffInsert})
	}

	marks := map[int]editor.RowMarkState{}
	setMark := func(row int, f func(*editor.RowMarkState)) {
		if row < 0 || row >= len(curLines) {
			return
		}
		state := marks[row]
		f(&state)
		marks[row] = state
	}

	newRow := 0
	for idx := 0; idx < len(ops); {
		if ops[idx].kind == diffEqual {
			newRow++
			idx++
			continue
		}

		startNewRow := newRow
		delCount, insCount := 0, 0
		for idx < len(ops) && ops[idx].kind != diffEqual {
			switch ops[idx].kind {
			case diffDelete:
				delCount++
			case diffInsert:
				insCount++
				newRow++
			}
			idx++
		}

		common := delCount
		if insCount < common {
			common = insCount
		}
		for r := 0; r < common; r++ {
			row := startNewRow + r
			setMark(row, func(s *editor.RowMarkState) { s.Updated = true })
		}
		for r := common; r < insCount; r++ {
			row := startNewRow + r
			setMark(row, func(s *editor.RowMarkState) { s.Inserted = true })
		}

		if delCount > common {
			anchor := startNewRow + common
			switch {
			case len(curLines) == 0:
				// No rows to attach a deleted marker to.
			case anchor < len(curLines):
				setMark(anchor, func(s *editor.RowMarkState) { s.DeletedAbove = true })
			default:
				setMark(len(curLines)-1, func(s *editor.RowMarkState) { s.DeletedBelow = true })
			}
		}
	}
	return marks
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
