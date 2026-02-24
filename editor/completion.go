package editor

import (
	"slices"

	"github.com/charmbracelet/bubbles/key"

	"github.com/iw2rmb/flourish/buffer"
)

const (
	defaultCompletionMaxVisibleRows = 8
	defaultCompletionMaxWidth       = 60
)

type CompletionSegment struct {
	Text      string
	StyleKey  string
	cellWidth int // cached terminal cell width; -1 means not computed
}

type CompletionItem struct {
	ID         string
	InsertText string
	Edits      []buffer.TextEdit

	Prefix []CompletionSegment
	Label  []CompletionSegment
	Detail []CompletionSegment

	StyleKey string
}

type CompletionState struct {
	Visible  bool
	Anchor   buffer.Pos
	Query    string
	Items    []CompletionItem
	Selected int

	VisibleIndices []int
}

type CompletionFilterContext struct {
	Query      string
	Items      []CompletionItem
	Cursor     buffer.Pos
	DocID      string
	DocVersion uint64

	// lowerCache holds pre-computed lowercased flattened text per item,
	// used by defaultCompletionFilter to avoid re-lowercasing on every keystroke.
	lowerCache []string
}

type CompletionFilterResult struct {
	VisibleIndices []int
	SelectedIndex  int
}

type CompletionFilter func(CompletionFilterContext) CompletionFilterResult

type CompletionInputMode uint8

const (
	CompletionInputQueryOnly CompletionInputMode = iota
	CompletionInputMutateDocument
)

type CompletionKeyMap struct {
	Trigger key.Binding
	Accept  key.Binding

	AcceptTab bool

	Dismiss  key.Binding
	Next     key.Binding
	Prev     key.Binding
	PageNext key.Binding
	PagePrev key.Binding
}

func (km CompletionKeyMap) isZero() bool {
	return !km.AcceptTab && allBindingsZero([]key.Binding{
		km.Trigger, km.Accept, km.Dismiss,
		km.Next, km.Prev, km.PageNext, km.PagePrev,
	})
}

func DefaultCompletionKeyMap() CompletionKeyMap {
	return CompletionKeyMap{
		Trigger: key.NewBinding(
			key.WithKeys("ctrl+space", "ctrl+@"),
			key.WithHelp("ctrl+space", "trigger completion"),
		),
		Accept:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept completion")),
		AcceptTab: true,
		Dismiss:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "dismiss completion")),
		Next:      key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "next completion")),
		Prev:      key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "prev completion")),
		PageNext:  key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "next completion page")),
		PagePrev:  key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "prev completion page")),
	}
}

type CompletionIntentKind uint8

const (
	IntentCompletionTrigger CompletionIntentKind = iota
	IntentCompletionNavigate
	IntentCompletionAccept
	IntentCompletionDismiss
	IntentCompletionQuery
)

type CompletionIntent struct {
	Kind    CompletionIntentKind
	Before  EditorState
	Payload any
}

type CompletionIntentBatch struct {
	Intents []CompletionIntent
}

type CompletionTriggerIntentPayload struct {
	Anchor buffer.Pos
}

type CompletionNavigateIntentPayload struct {
	Delta     int
	Selected  int
	ItemIndex int
}

type CompletionAcceptIntentPayload struct {
	ItemID       string
	ItemIndex    int
	VisibleIndex int
	InsertText   string
	Edits        []buffer.TextEdit
}

type CompletionDismissIntentPayload struct{}

type CompletionQueryIntentPayload struct {
	Query string
}

func normalizeCompletionInputMode(mode CompletionInputMode) CompletionInputMode {
	switch mode {
	case CompletionInputQueryOnly, CompletionInputMutateDocument:
		return mode
	default:
		return CompletionInputQueryOnly
	}
}

func normalizeCompletionKeyMap(km CompletionKeyMap) CompletionKeyMap {
	if km.isZero() {
		return DefaultCompletionKeyMap()
	}
	return km
}

func normalizeCompletionMaxVisibleRows(rows int) int {
	if rows <= 0 {
		return defaultCompletionMaxVisibleRows
	}
	return rows
}

func normalizeCompletionMaxWidth(width int) int {
	if width <= 0 {
		return defaultCompletionMaxWidth
	}
	return width
}

func normalizeCompletionState(state CompletionState) CompletionState {
	state = cloneCompletionState(state)
	state.VisibleIndices = sanitizeCompletionVisibleIndices(state.VisibleIndices, len(state.Items))

	if len(state.VisibleIndices) == 0 {
		state.Selected = 0
		return state
	}

	state.Selected = clampCompletionSelected(state.Selected, len(state.VisibleIndices))
	return state
}

func cloneCompletionState(state CompletionState) CompletionState {
	state.Items = cloneCompletionItems(state.Items)
	state.VisibleIndices = slices.Clone(state.VisibleIndices)
	return state
}

func cloneCompletionItems(items []CompletionItem) []CompletionItem {
	out := slices.Clone(items)
	for i := range out {
		out[i].Edits = slices.Clone(out[i].Edits)
		out[i].Prefix = slices.Clone(out[i].Prefix)
		out[i].Label = slices.Clone(out[i].Label)
		out[i].Detail = slices.Clone(out[i].Detail)
	}
	return out
}

func sanitizeCompletionVisibleIndices(indices []int, itemCount int) []int {
	if len(indices) == 0 || itemCount <= 0 {
		return nil
	}
	out := make([]int, 0, len(indices))
	seen := make(map[int]struct{}, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= itemCount {
			continue
		}
		if _, exists := seen[idx]; exists {
			continue
		}
		seen[idx] = struct{}{}
		out = append(out, idx)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func clampCompletionSelected(selected, visibleCount int) int {
	if visibleCount <= 0 {
		return 0
	}
	return clampInt(selected, 0, visibleCount-1)
}

func (m Model) CompletionState() CompletionState {
	return cloneCompletionState(m.completionState)
}

func (m Model) SetCompletionState(state CompletionState) Model {
	m.completionState = cloneCompletionState(state)
	m.completionLowerCache = nil // invalidate; recomputeCompletionFilter rebuilds
	m.recomputeCompletionFilter(&m.completionState)
	return m
}

func (m Model) ClearCompletion() Model {
	m.completionState = CompletionState{}
	return m
}
