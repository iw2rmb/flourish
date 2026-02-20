package editor

import (
	"reflect"
	"testing"

	"github.com/iw2rmb/flourish/buffer"
)

func TestCompletionAPI_ExportedTypesCompile(t *testing.T) {
	t.Helper()

	var _ CompletionSegment
	var _ CompletionItem
	var _ CompletionState
	var _ CompletionFilterContext
	var _ CompletionFilterResult
	var _ CompletionKeyMap
	var _ CompletionIntent
	var _ CompletionIntentBatch
	var _ CompletionFilter = func(CompletionFilterContext) CompletionFilterResult {
		return CompletionFilterResult{}
	}

	_ = CompletionInputQueryOnly
	_ = CompletionInputMutateDocument
	_ = IntentCompletionTrigger
	_ = IntentCompletionNavigate
	_ = IntentCompletionAccept
	_ = IntentCompletionDismiss
	_ = IntentCompletionQuery
}

func TestCompletionConfig_DefaultNormalization(t *testing.T) {
	m := New(Config{})

	if got, want := m.cfg.CompletionInputMode, CompletionInputQueryOnly; got != want {
		t.Fatalf("completion input mode default: got %v, want %v", got, want)
	}
	if got, want := m.cfg.CompletionMaxVisibleRows, defaultCompletionMaxVisibleRows; got != want {
		t.Fatalf("completion max visible rows default: got %d, want %d", got, want)
	}
	if got, want := m.cfg.CompletionMaxWidth, defaultCompletionMaxWidth; got != want {
		t.Fatalf("completion max width default: got %d, want %d", got, want)
	}
	if reflect.DeepEqual(m.cfg.CompletionKeyMap, CompletionKeyMap{}) {
		t.Fatalf("completion keymap should be defaulted from zero value")
	}
	if !m.cfg.CompletionKeyMap.AcceptTab {
		t.Fatalf("completion keymap default should enable AcceptTab")
	}
}

func TestCompletionConfig_PreservesExplicitValues(t *testing.T) {
	km := DefaultCompletionKeyMap()
	km.AcceptTab = false

	m := New(Config{
		CompletionInputMode:      CompletionInputMutateDocument,
		CompletionMaxVisibleRows: 3,
		CompletionMaxWidth:       24,
		CompletionKeyMap:         km,
	})

	if got, want := m.cfg.CompletionInputMode, CompletionInputMutateDocument; got != want {
		t.Fatalf("completion input mode: got %v, want %v", got, want)
	}
	if got, want := m.cfg.CompletionMaxVisibleRows, 3; got != want {
		t.Fatalf("completion max visible rows: got %d, want %d", got, want)
	}
	if got, want := m.cfg.CompletionMaxWidth, 24; got != want {
		t.Fatalf("completion max width: got %d, want %d", got, want)
	}
	if got, want := m.cfg.CompletionKeyMap.AcceptTab, false; got != want {
		t.Fatalf("completion keymap AcceptTab: got %v, want %v", got, want)
	}
}

func TestModelCompletionState_SetGetClearAndClamping(t *testing.T) {
	m := New(Config{})

	state := CompletionState{
		Visible: true,
		Anchor:  buffer.Pos{Row: 1, GraphemeCol: 2},
		Query:   "ab",
		Items: []CompletionItem{
			{
				ID:         "item-0",
				InsertText: "alpha",
				Edits: []buffer.TextEdit{
					{
						Range: buffer.Range{
							Start: buffer.Pos{Row: 1, GraphemeCol: 2},
							End:   buffer.Pos{Row: 1, GraphemeCol: 2},
						},
						Text: "alpha",
					},
				},
				Prefix:   []CompletionSegment{{Text: "f", StyleKey: "kind"}},
				Label:    []CompletionSegment{{Text: "alpha", StyleKey: "label"}},
				Detail:   []CompletionSegment{{Text: "detail", StyleKey: "detail"}},
				StyleKey: "row",
			},
			{
				ID:         "item-1",
				InsertText: "beta",
				Label:      []CompletionSegment{{Text: "beta"}},
			},
		},
		VisibleIndices: []int{-1, 1, 1, 2, 0},
		Selected:       99,
	}

	m = m.SetCompletionState(state)
	got := m.CompletionState()

	if got.Visible != state.Visible || got.Anchor != state.Anchor || got.Query != state.Query {
		t.Fatalf("state fields mismatch: got %+v, want %+v", got, state)
	}
	if gotN, wantN := len(got.VisibleIndices), 2; gotN != wantN {
		t.Fatalf("visible index count: got %d, want %d", gotN, wantN)
	}
	if got.VisibleIndices[0] != 1 || got.VisibleIndices[1] != 0 {
		t.Fatalf("visible indices sanitization: got %v, want [1 0]", got.VisibleIndices)
	}
	if got, want := got.Selected, 1; got != want {
		t.Fatalf("selected index clamping: got %d, want %d", got, want)
	}

	state.Items[0].Label[0].Text = "mutated"
	state.Items[0].Edits[0].Text = "mutated"
	state.VisibleIndices[1] = 0
	afterInputMutation := m.CompletionState()
	if got, want := afterInputMutation.Items[0].Label[0].Text, "alpha"; got != want {
		t.Fatalf("model state should not alias input state label: got %q, want %q", got, want)
	}
	if got, want := afterInputMutation.Items[0].Edits[0].Text, "alpha"; got != want {
		t.Fatalf("model state should not alias input state edits: got %q, want %q", got, want)
	}

	got.Items[0].Label[0].Text = "snapshot-mutation"
	got.VisibleIndices[0] = 0
	afterOutputMutation := m.CompletionState()
	if got, want := afterOutputMutation.Items[0].Label[0].Text, "alpha"; got != want {
		t.Fatalf("completion getter should return cloned label slice: got %q, want %q", got, want)
	}
	if got, want := afterOutputMutation.VisibleIndices[0], 1; got != want {
		t.Fatalf("completion getter should return cloned visible indices: got %d, want %d", got, want)
	}

	m = m.SetCompletionState(CompletionState{
		Visible:  true,
		Items:    []CompletionItem{{ID: "only"}},
		Selected: 5,
	})
	got = m.CompletionState()
	if got, want := got.Selected, 0; got != want {
		t.Fatalf("selected should reset when no visible indices are set: got %d, want %d", got, want)
	}

	m = m.ClearCompletion()
	got = m.CompletionState()
	if !reflect.DeepEqual(got, CompletionState{}) {
		t.Fatalf("clear completion state: got %+v, want zero value", got)
	}
}
