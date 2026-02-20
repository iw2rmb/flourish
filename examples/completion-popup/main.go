package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iw2rmb/flourish/buffer"
	"github.com/iw2rmb/flourish/editor"
)

type hostState struct {
	lastCompletion string
}

type model struct {
	editor editor.Model
	items  []editor.CompletionItem
	host   *hostState
}

func newModel() model {
	host := &hostState{}
	km := editor.DefaultCompletionKeyMap()
	km.AcceptTab = false

	cfg := editor.Config{
		Text: strings.Join([]string{
			"Completion popup example",
			"ctrl+space: open popup, enter: accept, esc: dismiss",
			"tab inserts literal tab (completion tab-accept disabled)",
			"",
			"type profile struct {",
			"\tname string",
			"\tage  int",
			"}",
			"",
			"func build() {",
			"\tpro",
			"}",
			"",
			"ctrl+q quits",
		}, "\n"),
		Gutter:                   editor.LineNumberGutter(),
		Style:                    editor.DefaultStyle(),
		CompletionKeyMap:         km,
		CompletionInputMode:      editor.CompletionInputMutateDocument,
		CompletionMaxVisibleRows: 6,
		CompletionMaxWidth:       40,
		CompletionFilter:         completionFilter,
		CompletionStyleForKey:    completionStyleForKey,
		OnCompletionIntent: func(batch editor.CompletionIntentBatch) {
			if len(batch.Intents) == 0 {
				return
			}
			in := batch.Intents[len(batch.Intents)-1]
			host.lastCompletion = completionIntentSummary(in)
		},
	}

	m := model{
		editor: editor.New(cfg),
		items:  completionItems(),
		host:   host,
	}
	m.editor.Buffer().SetCursor(buffer.Pos{Row: 10, GraphemeCol: 4})
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
	m.syncCompletionItems()
	return m, cmd
}

func (m model) View() string {
	state := m.editor.CompletionState()
	status := "completion hidden"
	if state.Visible {
		status = fmt.Sprintf("completion visible | query=%q | selected=%d", state.Query, state.Selected)
	}
	return strings.Join([]string{
		"Completion popup example",
		status,
		"last completion intent: " + noneIfEmpty(m.host.lastCompletion),
		m.editor.View(),
	}, "\n")
}

func (m *model) syncCompletionItems() {
	state := m.editor.CompletionState()
	if !state.Visible {
		return
	}
	state.Items = append([]editor.CompletionItem(nil), m.items...)
	m.editor = m.editor.SetCompletionState(state)
}

func editorHeight(total int) int {
	h := total - 3
	if h < 0 {
		return 0
	}
	return h
}

func completionItems() []editor.CompletionItem {
	return []editor.CompletionItem{
		{
			ID:         "kw-prop",
			InsertText: "property",
			Prefix:     []editor.CompletionSegment{{Text: "kw", StyleKey: "kind.keyword"}},
			Label:      []editor.CompletionSegment{{Text: "property"}},
			Detail:     []editor.CompletionSegment{{Text: "keyword", StyleKey: "detail.meta"}},
			StyleKey:   "item.default",
		},
		{
			ID:         "fn-println",
			InsertText: "println()",
			Prefix:     []editor.CompletionSegment{{Text: "fn", StyleKey: "kind.function"}},
			Label:      []editor.CompletionSegment{{Text: "println"}},
			Detail:     []editor.CompletionSegment{{Text: "stdout", StyleKey: "detail.meta"}},
			StyleKey:   "item.default",
		},
		{
			ID:         "type-profile",
			InsertText: "profile{}",
			Prefix:     []editor.CompletionSegment{{Text: "tp", StyleKey: "kind.type"}},
			Label:      []editor.CompletionSegment{{Text: "profile"}},
			Detail:     []editor.CompletionSegment{{Text: "type", StyleKey: "detail.meta"}},
			StyleKey:   "item.default",
		},
		{
			ID:         "var-project",
			InsertText: "project",
			Prefix:     []editor.CompletionSegment{{Text: "vr", StyleKey: "kind.variable"}},
			Label:      []editor.CompletionSegment{{Text: "project"}},
			Detail:     []editor.CompletionSegment{{Text: "string", StyleKey: "detail.meta"}},
			StyleKey:   "item.default",
		},
	}
}

func completionStyleForKey(key string) (lipgloss.Style, bool) {
	switch key {
	case "kind.keyword":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")), true
	case "kind.function":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("81")), true
	case "kind.type":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("141")), true
	case "kind.variable":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("114")), true
	case "detail.meta":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")), true
	case "item.default":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252")), true
	default:
		return lipgloss.Style{}, false
	}
}

func completionFilter(ctx editor.CompletionFilterContext) editor.CompletionFilterResult {
	query := strings.ToLower(strings.TrimSpace(ctx.Query))
	if len(ctx.Items) == 0 {
		return editor.CompletionFilterResult{}
	}
	if query == "" {
		indices := make([]int, len(ctx.Items))
		for i := range ctx.Items {
			indices[i] = i
		}
		return editor.CompletionFilterResult{VisibleIndices: indices, SelectedIndex: 0}
	}

	type ranked struct {
		index int
		score int
		label string
	}

	rankedItems := make([]ranked, 0, len(ctx.Items))
	for i, item := range ctx.Items {
		label := strings.ToLower(flattenSegments(item.Label))
		flat := strings.ToLower(flattenItem(item))
		switch {
		case strings.HasPrefix(label, query):
			rankedItems = append(rankedItems, ranked{index: i, score: 0, label: label})
		case strings.Contains(label, query):
			rankedItems = append(rankedItems, ranked{index: i, score: 1, label: label})
		case strings.Contains(flat, query):
			rankedItems = append(rankedItems, ranked{index: i, score: 2, label: label})
		}
	}

	sort.SliceStable(rankedItems, func(i, j int) bool {
		if rankedItems[i].score != rankedItems[j].score {
			return rankedItems[i].score < rankedItems[j].score
		}
		if rankedItems[i].label != rankedItems[j].label {
			return rankedItems[i].label < rankedItems[j].label
		}
		return rankedItems[i].index < rankedItems[j].index
	})

	indices := make([]int, 0, len(rankedItems))
	for _, item := range rankedItems {
		indices = append(indices, item.index)
	}
	return editor.CompletionFilterResult{VisibleIndices: indices, SelectedIndex: 0}
}

func flattenItem(item editor.CompletionItem) string {
	return strings.Join([]string{
		flattenSegments(item.Prefix),
		flattenSegments(item.Label),
		flattenSegments(item.Detail),
	}, " ")
}

func flattenSegments(segments []editor.CompletionSegment) string {
	if len(segments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(segments))
	for _, seg := range segments {
		if seg.Text != "" {
			parts = append(parts, seg.Text)
		}
	}
	return strings.Join(parts, " ")
}

func completionIntentSummary(in editor.CompletionIntent) string {
	switch payload := in.Payload.(type) {
	case editor.CompletionTriggerIntentPayload:
		return fmt.Sprintf("trigger anchor=(%d,%d)", payload.Anchor.Row, payload.Anchor.GraphemeCol)
	case editor.CompletionNavigateIntentPayload:
		return fmt.Sprintf("navigate delta=%d selected=%d item=%d", payload.Delta, payload.Selected, payload.ItemIndex)
	case editor.CompletionAcceptIntentPayload:
		return fmt.Sprintf("accept item=%q edits=%d", payload.ItemID, len(payload.Edits))
	case editor.CompletionDismissIntentPayload:
		return "dismiss"
	case editor.CompletionQueryIntentPayload:
		return fmt.Sprintf("query %q", payload.Query)
	default:
		return "unknown"
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
