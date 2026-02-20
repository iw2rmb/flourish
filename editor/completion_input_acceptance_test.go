package editor

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

func TestCompletionInput_KeyRoutingVisibleVsHidden(t *testing.T) {
	hidden := New(Config{Text: "a\nb\nc"})
	hidden, _ = hidden.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got, want := hidden.buf.Cursor(), (buffer.Pos{Row: 1, GraphemeCol: 0}); got != want {
		t.Fatalf("hidden popup down key should move cursor: got %v, want %v", got, want)
	}

	visible := New(Config{Text: "a\nb\nc"})
	visible = visible.SetCompletionState(CompletionState{
		Visible:        true,
		Items:          []CompletionItem{{ID: "0"}, {ID: "1"}},
		VisibleIndices: []int{0, 1},
		Selected:       0,
	})
	visible, _ = visible.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got, want := visible.buf.Cursor(), (buffer.Pos{Row: 0, GraphemeCol: 0}); got != want {
		t.Fatalf("visible popup down key should not move cursor: got %v, want %v", got, want)
	}
	if got, want := visible.CompletionState().Selected, 1; got != want {
		t.Fatalf("visible popup down key should move completion selection: got %d, want %d", got, want)
	}
}

func TestCompletionInput_TriggerOpensAtCursor(t *testing.T) {
	km := DefaultCompletionKeyMap()
	km.Trigger = key.NewBinding(key.WithKeys("down"))

	m := New(Config{
		Text:             "ab",
		CompletionKeyMap: km,
	})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	state := m.CompletionState()
	if !state.Visible {
		t.Fatalf("completion popup should open on trigger key")
	}
	if got, want := state.Anchor, (buffer.Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("completion anchor: got %v, want %v", got, want)
	}
	if got, want := state.Query, ""; got != want {
		t.Fatalf("completion query after trigger: got %q, want %q", got, want)
	}
	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("trigger key should not mutate text: got %q, want %q", got, want)
	}
}

func TestCompletionInput_DefaultTriggerMatchesCtrlSpaceAlias(t *testing.T) {
	m := New(Config{Text: "ab"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlAt})

	state := m.CompletionState()
	if !state.Visible {
		t.Fatalf("completion popup should open on ctrl+space alias ctrl+@")
	}
	if got, want := state.Anchor, (buffer.Pos{Row: 0, GraphemeCol: 0}); got != want {
		t.Fatalf("completion anchor: got %v, want %v", got, want)
	}
}

func TestCompletionAccept_AppliesItemEditsAndClearsPopup(t *testing.T) {
	m := New(Config{Text: "hello"})
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Items: []CompletionItem{
			{
				ID: "edit",
				Edits: []buffer.TextEdit{{
					Range: buffer.Range{
						Start: buffer.Pos{Row: 0, GraphemeCol: 1},
						End:   buffer.Pos{Row: 0, GraphemeCol: 4},
					},
					Text: "XYZ",
				}},
			},
		},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, want := m.buf.Text(), "hXYZo"; got != want {
		t.Fatalf("accept should apply item edits: got %q, want %q", got, want)
	}
	if got := m.CompletionState(); !reflect.DeepEqual(got, CompletionState{}) {
		t.Fatalf("accept should clear popup state: got %+v", got)
	}
}

func TestCompletionAccept_FallsBackToInsertTextAtAnchor(t *testing.T) {
	m := New(Config{Text: "abcd"})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 4})
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Anchor:  buffer.Pos{Row: 0, GraphemeCol: 1},
		Items: []CompletionItem{
			{ID: "insert", InsertText: "ZZ"},
		},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, want := m.buf.Text(), "aZZbcd"; got != want {
		t.Fatalf("fallback insert should use completion anchor: got %q, want %q", got, want)
	}
}

func TestCompletionAccept_AcceptTabGate(t *testing.T) {
	t.Run("AcceptTab=true accepts selected item", func(t *testing.T) {
		m := New(Config{Text: "ab"})
		m = m.SetCompletionState(CompletionState{
			Visible: true,
			Items: []CompletionItem{
				{ID: "insert", InsertText: "X"},
			},
			VisibleIndices: []int{0},
		})

		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		if got, want := m.buf.Text(), "Xab"; got != want {
			t.Fatalf("tab should accept completion when enabled: got %q, want %q", got, want)
		}
	})

	t.Run("AcceptTab=false falls through to normal tab handling", func(t *testing.T) {
		km := DefaultCompletionKeyMap()
		km.AcceptTab = false
		m := New(Config{
			Text:             "ab",
			CompletionKeyMap: km,
		})
		m = m.SetCompletionState(CompletionState{
			Visible: true,
			Items: []CompletionItem{
				{ID: "insert", InsertText: "X"},
			},
			VisibleIndices: []int{0},
		})

		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		if got, want := m.buf.Text(), "\tab"; got != want {
			t.Fatalf("tab should fall through when AcceptTab=false: got %q, want %q", got, want)
		}
		if m.CompletionState().Visible {
			t.Fatalf("popup should close when tab moves cursor outside anchor token")
		}
	})
}

func TestCompletionInput_QueryOnlyModeUpdatesQueryWithoutDocMutation(t *testing.T) {
	m := New(Config{Text: "ab"})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Items:          []CompletionItem{{ID: "0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("query-only typing/backspace should not mutate text: got %q, want %q", got, want)
	}
	state := m.CompletionState()
	if got, want := state.Query, "x"; got != want {
		t.Fatalf("query-only mode query: got %q, want %q", got, want)
	}
	if !state.Visible {
		t.Fatalf("query-only input should keep popup visible")
	}
}

func TestCompletionInput_MutateDocumentModeMutatesTextAndRecomputesQuery(t *testing.T) {
	m := New(Config{
		Text:                "ab",
		CompletionInputMode: CompletionInputMutateDocument,
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Anchor:         buffer.Pos{Row: 0, GraphemeCol: 1},
		Items:          []CompletionItem{{ID: "0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Z")})
	if got, want := m.buf.Text(), "aZb"; got != want {
		t.Fatalf("mutate-document typing should mutate text: got %q, want %q", got, want)
	}
	if got, want := m.CompletionState().Query, "Z"; got != want {
		t.Fatalf("mutate-document typing should recompute query: got %q, want %q", got, want)
	}
	if !m.CompletionState().Visible {
		t.Fatalf("mutate-document typing should keep popup visible")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("mutate-document backspace should mutate text: got %q, want %q", got, want)
	}
	if got, want := m.CompletionState().Query, ""; got != want {
		t.Fatalf("mutate-document backspace should recompute query: got %q, want %q", got, want)
	}
	if !m.CompletionState().Visible {
		t.Fatalf("mutate-document backspace should keep popup visible")
	}
}

func TestCompletionInput_MutateDocumentModeAnchorInvalidationResetsQuery(t *testing.T) {
	m := New(Config{
		Text:                "ab",
		CompletionInputMode: CompletionInputMutateDocument,
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Anchor:         buffer.Pos{Row: 0, GraphemeCol: 1},
		Query:          "x",
		Items:          []CompletionItem{{ID: "0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got, want := m.CompletionState().Query, ""; got != want {
		t.Fatalf("query should reset when cursor moves before anchor: got %q, want %q", got, want)
	}
}

func TestCompletionInput_ReadOnlyMutateModeBehavesAsQueryOnly(t *testing.T) {
	m := New(Config{
		Text:                "ab",
		ReadOnly:            true,
		CompletionInputMode: CompletionInputMutateDocument,
	})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Items:          []CompletionItem{{ID: "0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("read-only mutate mode should not mutate text: got %q, want %q", got, want)
	}
	if got, want := m.CompletionState().Query, "x"; got != want {
		t.Fatalf("read-only mutate mode should update query: got %q, want %q", got, want)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got, want := m.CompletionState().Query, ""; got != want {
		t.Fatalf("read-only mutate mode backspace should update query: got %q, want %q", got, want)
	}
}

func TestCompletionAccept_ReadOnlySuppressesLocalMutation(t *testing.T) {
	m := New(Config{
		Text:     "ab",
		ReadOnly: true,
	})
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Items: []CompletionItem{
			{ID: "insert", InsertText: "X"},
		},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("read-only accept should not mutate text: got %q, want %q", got, want)
	}
	if !m.CompletionState().Visible {
		t.Fatalf("read-only accept should not clear popup without local apply")
	}
}

func TestCompletionCursorMoveWithinAnchorTokenKeepsPopupVisible(t *testing.T) {
	m := New(Config{Text: "pro x"})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 3})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Anchor:         buffer.Pos{Row: 0, GraphemeCol: 3},
		Items:          []CompletionItem{{ID: "item-0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if !m.CompletionState().Visible {
		t.Fatalf("popup should stay visible while cursor remains within anchor token")
	}
}

func TestCompletionCursorMoveOutsideAnchorTokenClosesPopup(t *testing.T) {
	m := New(Config{Text: "pro x"})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 3})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Anchor:         buffer.Pos{Row: 0, GraphemeCol: 3},
		Items:          []CompletionItem{{ID: "item-0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.CompletionState().Visible {
		t.Fatalf("popup should close when cursor leaves anchor token on same row")
	}
}

func TestCompletionCursorMoveToDifferentRowClosesPopup(t *testing.T) {
	m := New(Config{Text: "pro\nx"})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 3})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Anchor:         buffer.Pos{Row: 0, GraphemeCol: 3},
		Items:          []CompletionItem{{ID: "item-0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.CompletionState().Visible {
		t.Fatalf("popup should close when cursor leaves anchor row")
	}
}
