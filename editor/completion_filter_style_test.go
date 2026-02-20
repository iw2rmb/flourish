package editor

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestCompletionFilter_DefaultFilter_MixedCaseStableOrder(t *testing.T) {
	m := New(Config{})
	m = m.SetCompletionState(CompletionState{
		Query: "al",
		Items: []CompletionItem{
			{ID: "0", Label: []CompletionSegment{{Text: "Alpha"}}},
			{ID: "1", Prefix: []CompletionSegment{{Text: "beta"}}},
			{ID: "2", Detail: []CompletionSegment{{Text: "xALy"}}},
			{ID: "3", Label: []CompletionSegment{{Text: "zzz"}}},
		},
	})

	state := m.CompletionState()
	if got, want := state.VisibleIndices, []int{0, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("default filter visible indices: got %v, want %v", got, want)
	}
	if got, want := state.Selected, 0; got != want {
		t.Fatalf("default filter selected: got %d, want %d", got, want)
	}
}

func TestCompletionFilter_Callback_SanitizesIndicesAndClampsSelected(t *testing.T) {
	m := New(Config{
		CompletionFilter: func(ctx CompletionFilterContext) CompletionFilterResult {
			return CompletionFilterResult{
				VisibleIndices: []int{-1, 2, 2, 1, 9},
				SelectedIndex:  99,
			}
		},
	})
	m = m.SetCompletionState(CompletionState{
		Items: []CompletionItem{{ID: "0"}, {ID: "1"}, {ID: "2"}},
	})

	state := m.CompletionState()
	if got, want := state.VisibleIndices, []int{2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("callback filter visible indices: got %v, want %v", got, want)
	}
	if got, want := state.Selected, 1; got != want {
		t.Fatalf("callback filter selected clamp: got %d, want %d", got, want)
	}
}

func TestCompletionFilter_CallbackInvokedOnQueryItemsAndContextChanges(t *testing.T) {
	callCount := 0
	queries := make([]string, 0, 4)

	m := New(Config{
		Text: "ab",
		CompletionFilter: func(ctx CompletionFilterContext) CompletionFilterResult {
			callCount++
			queries = append(queries, ctx.Query)
			visible := make([]int, len(ctx.Items))
			for i := range ctx.Items {
				visible[i] = i
			}
			return CompletionFilterResult{VisibleIndices: visible, SelectedIndex: 0}
		},
	})

	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Items:   []CompletionItem{{ID: "0"}, {ID: "1"}},
	})
	if got, want := callCount, 1; got != want {
		t.Fatalf("set completion should run filter once: got %d, want %d", got, want)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got, want := callCount, 1; got != want {
		t.Fatalf("selection navigation should not rerun filter: got %d, want %d", got, want)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if got, want := callCount, 2; got != want {
		t.Fatalf("query change should rerun filter: got %d, want %d", got, want)
	}
	if got, want := queries[1], "x"; got != want {
		t.Fatalf("query change callback context: got %q, want %q", got, want)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got, want := callCount, 3; got != want {
		t.Fatalf("cursor change should rerun filter: got %d, want %d", got, want)
	}

	m.buf.InsertText("q")
	m, _ = m.Update(struct{}{})
	if got, want := callCount, 4; got != want {
		t.Fatalf("doc version change should rerun filter: got %d, want %d", got, want)
	}
}

func TestCompletionStyle_ResolutionAndSelectedBase(t *testing.T) {
	st := Style{
		CompletionItem:     lipgloss.NewStyle().PaddingRight(1),
		CompletionSelected: lipgloss.NewStyle().PaddingLeft(1),
	}

	baseSelected := completionRowBaseStyle(st, true)
	if got, want := stripANSI(baseSelected.Render("x")), " x"; got != want {
		t.Fatalf("selected base style should use completion selected style: got %q, want %q", got, want)
	}

	styleForKey := func(key string) (lipgloss.Style, bool) {
		switch key {
		case "item":
			return lipgloss.NewStyle().PaddingLeft(1), true
		case "seg":
			return lipgloss.NewStyle().PaddingLeft(2), true
		default:
			return lipgloss.Style{}, false
		}
	}

	item := CompletionItem{StyleKey: "item"}
	seg := CompletionSegment{Text: "x", StyleKey: "seg"}
	if got, want := stripANSI(resolveCompletionSegmentStyle(st.CompletionItem, styleForKey, item, seg).Render("x")), "  x"; got != want {
		t.Fatalf("segment style should win over item style: got %q, want %q", got, want)
	}

	seg.StyleKey = ""
	if got, want := stripANSI(resolveCompletionSegmentStyle(st.CompletionItem, styleForKey, item, seg).Render("x")), " x"; got != want {
		t.Fatalf("item style should apply when segment key missing: got %q, want %q", got, want)
	}

	item.StyleKey = ""
	if got, want := stripANSI(resolveCompletionSegmentStyle(st.CompletionItem, styleForKey, item, seg).Render("x")), "x "; got != want {
		t.Fatalf("default style should apply when keys are unresolved: got %q, want %q", got, want)
	}
}

func TestCompletionStyle_TruncateSegments_OrderAndTailHandling(t *testing.T) {
	segments := []CompletionSegment{
		{Text: "ab", StyleKey: "a"},
		{Text: "cd", StyleKey: "b"},
	}
	if got, want := truncateCompletionSegments(segments, 3), []CompletionSegment{
		{Text: "ab", StyleKey: "a"},
		{Text: "c", StyleKey: "b"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascii truncation: got %v, want %v", got, want)
	}

	wide := []CompletionSegment{
		{Text: "界", StyleKey: "w"},
		{Text: "x", StyleKey: "x"},
	}
	if got, want := truncateCompletionSegments(wide, 1), []CompletionSegment{
		{Text: " ", StyleKey: "w"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wide tail truncation should keep style key and width: got %v, want %v", got, want)
	}
}
