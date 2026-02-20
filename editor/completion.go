package editor

import (
	"reflect"

	"github.com/charmbracelet/bubbles/key"

	"github.com/iw2rmb/flourish/buffer"
)

const (
	defaultCompletionMaxVisibleRows = 8
	defaultCompletionMaxWidth       = 60
)

type CompletionSegment struct {
	Text     string
	StyleKey string
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

func DefaultCompletionKeyMap() CompletionKeyMap {
	return CompletionKeyMap{
		Trigger:   key.NewBinding(key.WithKeys("ctrl+space"), key.WithHelp("ctrl+space", "trigger completion")),
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

func normalizeCompletionInputMode(mode CompletionInputMode) CompletionInputMode {
	switch mode {
	case CompletionInputQueryOnly, CompletionInputMutateDocument:
		return mode
	default:
		return CompletionInputQueryOnly
	}
}

func normalizeCompletionKeyMap(km CompletionKeyMap) CompletionKeyMap {
	if reflect.DeepEqual(km, CompletionKeyMap{}) {
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

	state.Selected = clampInt(state.Selected, 0, len(state.VisibleIndices)-1)
	return state
}

func cloneCompletionState(state CompletionState) CompletionState {
	state.Items = cloneCompletionItems(state.Items)
	if len(state.VisibleIndices) == 0 {
		state.VisibleIndices = nil
	} else {
		state.VisibleIndices = append([]int(nil), state.VisibleIndices...)
	}
	return state
}

func cloneCompletionItems(items []CompletionItem) []CompletionItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]CompletionItem, len(items))
	copy(out, items)
	for i := range out {
		out[i].Edits = cloneTextEdits(out[i].Edits)
		out[i].Prefix = cloneCompletionSegments(out[i].Prefix)
		out[i].Label = cloneCompletionSegments(out[i].Label)
		out[i].Detail = cloneCompletionSegments(out[i].Detail)
	}
	return out
}

func cloneCompletionSegments(segments []CompletionSegment) []CompletionSegment {
	if len(segments) == 0 {
		return nil
	}
	out := make([]CompletionSegment, len(segments))
	copy(out, segments)
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
