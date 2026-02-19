package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/editor"
)

type intentState struct {
	intentCount int
	changeCount int
	lastIntent  string
	lastPayload string
}

func (s *intentState) handleIntent(batch editor.IntentBatch) editor.IntentDecision {
	s.intentCount += len(batch.Intents)
	if len(batch.Intents) > 0 {
		last := batch.Intents[len(batch.Intents)-1]
		s.lastIntent = intentKindString(last.Kind)
		s.lastPayload = summarizePayload(last)
	}

	// Example policy: apply locally for navigation/selection, but leave text
	// mutations to the host's remote sync path.
	applyLocally := true
	for _, in := range batch.Intents {
		switch in.Kind {
		case editor.IntentInsert, editor.IntentDelete, editor.IntentUndo, editor.IntentRedo, editor.IntentPaste:
			applyLocally = false
		}
	}
	return editor.IntentDecision{ApplyLocally: applyLocally}
}

func (s *intentState) handleChange(editor.ChangeEvent) {
	s.changeCount++
}

type model struct {
	editor  editor.Model
	intents *intentState
}

func newModel() model {
	state := &intentState{}
	cfg := editor.Config{
		Text: strings.Join([]string{
			"Intent mode example",
			"Navigation/select updates apply locally.",
			"Typing/deletes/paste are emitted and not applied locally.",
			"Ctrl+Q quits.",
		}, "\n"),
		Gutter:       editor.LineNumberGutter(),
		Style:        editor.DefaultStyle(),
		MutationMode: editor.EmitIntentsAndMutate,
		OnIntent:     state.handleIntent,
		OnChange:     state.handleChange,
	}
	return model{editor: editor.New(cfg), intents: state}
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
	lines := []string{
		"",
		"Intent status:",
		fmt.Sprintf("intents: %d", m.intents.intentCount),
		fmt.Sprintf("local changes: %d", m.intents.changeCount),
		fmt.Sprintf("last intent: %s", noneIfEmpty(m.intents.lastIntent)),
		fmt.Sprintf("last payload: %s", noneIfEmpty(m.intents.lastPayload)),
	}
	return m.editor.View() + strings.Join(lines, "\n")
}

func editorHeight(total int) int {
	h := total - 7
	if h < 0 {
		return 0
	}
	return h
}

func intentKindString(kind editor.IntentKind) string {
	switch kind {
	case editor.IntentInsert:
		return "insert"
	case editor.IntentDelete:
		return "delete"
	case editor.IntentMove:
		return "move"
	case editor.IntentSelect:
		return "select"
	case editor.IntentUndo:
		return "undo"
	case editor.IntentRedo:
		return "redo"
	case editor.IntentPaste:
		return "paste"
	default:
		return "unknown"
	}
}

func summarizePayload(in editor.Intent) string {
	switch p := in.Payload.(type) {
	case editor.InsertIntentPayload:
		return fmt.Sprintf("text=%q edits=%d", p.Text, len(p.Edits))
	case editor.DeleteIntentPayload:
		return fmt.Sprintf("direction=%d", p.Direction)
	case editor.MoveIntentPayload:
		return fmt.Sprintf("move=(unit=%d dir=%d extend=%v)", p.Move.Unit, p.Move.Dir, p.Move.Extend)
	case editor.SelectIntentPayload:
		return fmt.Sprintf("move=(unit=%d dir=%d extend=%v)", p.Move.Unit, p.Move.Dir, p.Move.Extend)
	case editor.PasteIntentPayload:
		return fmt.Sprintf("text=%q", p.Text)
	default:
		return "-"
	}
}

func noneIfEmpty(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
