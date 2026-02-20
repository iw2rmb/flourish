package editor

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

func TestCompletionIntent_TriggerEmitsAndOpensInEmitIntentsOnly(t *testing.T) {
	km := DefaultCompletionKeyMap()
	km.Trigger = key.NewBinding(key.WithKeys("f2"))

	var batches []CompletionIntentBatch
	onIntentCalls := 0
	m := New(Config{
		Text:             "ab",
		CompletionKeyMap: km,
		MutationMode:     EmitIntentsOnly,
		OnCompletionIntent: func(batch CompletionIntentBatch) {
			batches = append(batches, batch)
		},
		OnIntent: func(IntentBatch) IntentDecision {
			onIntentCalls++
			return IntentDecision{ApplyLocally: true}
		},
	})
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF2})

	if got, want := len(batches), 1; got != want {
		t.Fatalf("completion intent batch count: got %d, want %d", got, want)
	}
	if got, want := len(batches[0].Intents), 1; got != want {
		t.Fatalf("completion intent count: got %d, want %d", got, want)
	}
	in := batches[0].Intents[0]
	if got, want := in.Kind, IntentCompletionTrigger; got != want {
		t.Fatalf("completion intent kind: got %v, want %v", got, want)
	}
	payload, ok := in.Payload.(CompletionTriggerIntentPayload)
	if !ok {
		t.Fatalf("trigger payload type: got %T", in.Payload)
	}
	if got, want := payload.Anchor, (buffer.Pos{Row: 0, GraphemeCol: 1}); got != want {
		t.Fatalf("trigger payload anchor: got %v, want %v", got, want)
	}
	if got := m.CompletionState(); !got.Visible {
		t.Fatalf("trigger should open completion in EmitIntentsOnly mode")
	}
	if got, want := onIntentCalls, 0; got != want {
		t.Fatalf("document intent callback calls: got %d, want %d", got, want)
	}
}

func TestCompletionIntent_NavigateAndDismissEmitAndUpdateStateInEmitIntentsOnly(t *testing.T) {
	var completionIntents []CompletionIntent
	documentIntentCalls := 0

	m := New(Config{
		Text:         "ab",
		MutationMode: EmitIntentsOnly,
		OnCompletionIntent: func(batch CompletionIntentBatch) {
			completionIntents = append(completionIntents, batch.Intents...)
		},
		OnIntent: func(IntentBatch) IntentDecision {
			documentIntentCalls++
			return IntentDecision{ApplyLocally: true}
		},
	})
	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Items: []CompletionItem{
			{ID: "0", InsertText: "x"},
			{ID: "1", InsertText: "y"},
		},
		VisibleIndices: []int{0, 1},
		Selected:       0,
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got, want := m.CompletionState().Selected, 1; got != want {
		t.Fatalf("selected after navigate: got %d, want %d", got, want)
	}
	if got, want := len(completionIntents), 1; got != want {
		t.Fatalf("completion intent count after navigate: got %d, want %d", got, want)
	}
	if got, want := completionIntents[0].Kind, IntentCompletionNavigate; got != want {
		t.Fatalf("navigate intent kind: got %v, want %v", got, want)
	}
	navPayload, ok := completionIntents[0].Payload.(CompletionNavigateIntentPayload)
	if !ok {
		t.Fatalf("navigate payload type: got %T", completionIntents[0].Payload)
	}
	if got, want := navPayload.Delta, 1; got != want {
		t.Fatalf("navigate payload delta: got %d, want %d", got, want)
	}
	if got, want := navPayload.Selected, 1; got != want {
		t.Fatalf("navigate payload selected: got %d, want %d", got, want)
	}
	if got, want := navPayload.ItemIndex, 1; got != want {
		t.Fatalf("navigate payload item index: got %d, want %d", got, want)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := m.CompletionState(); got.Visible {
		t.Fatalf("dismiss should clear completion state")
	}
	if got, want := len(completionIntents), 2; got != want {
		t.Fatalf("completion intent count after dismiss: got %d, want %d", got, want)
	}
	if got, want := completionIntents[1].Kind, IntentCompletionDismiss; got != want {
		t.Fatalf("dismiss intent kind: got %v, want %v", got, want)
	}
	if got, want := documentIntentCalls, 0; got != want {
		t.Fatalf("document intent callback calls: got %d, want %d", got, want)
	}
}

func TestCompletionIntent_AcceptMutationModeParity(t *testing.T) {
	tests := []struct {
		name              string
		mode              MutationMode
		decision          IntentDecision
		wantText          string
		wantPopupVisible  bool
		wantDocIntentCall bool
	}{
		{
			name:              "MutateInEditor",
			mode:              MutateInEditor,
			decision:          IntentDecision{ApplyLocally: false},
			wantText:          "aXb",
			wantPopupVisible:  false,
			wantDocIntentCall: false,
		},
		{
			name:              "EmitIntentsOnly",
			mode:              EmitIntentsOnly,
			decision:          IntentDecision{ApplyLocally: true},
			wantText:          "ab",
			wantPopupVisible:  true,
			wantDocIntentCall: true,
		},
		{
			name:              "EmitIntentsAndMutate apply=false",
			mode:              EmitIntentsAndMutate,
			decision:          IntentDecision{ApplyLocally: false},
			wantText:          "ab",
			wantPopupVisible:  true,
			wantDocIntentCall: true,
		},
		{
			name:              "EmitIntentsAndMutate apply=true",
			mode:              EmitIntentsAndMutate,
			decision:          IntentDecision{ApplyLocally: true},
			wantText:          "aXb",
			wantPopupVisible:  false,
			wantDocIntentCall: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var completionIntents []CompletionIntent
			var documentIntents []Intent
			documentIntentCalls := 0

			m := New(Config{
				Text:         "ab",
				MutationMode: tc.mode,
				OnCompletionIntent: func(batch CompletionIntentBatch) {
					completionIntents = append(completionIntents, batch.Intents...)
				},
				OnIntent: func(batch IntentBatch) IntentDecision {
					documentIntentCalls++
					documentIntents = append(documentIntents, batch.Intents...)
					return tc.decision
				},
			})
			m = m.SetCompletionState(CompletionState{
				Visible: true,
				Anchor:  buffer.Pos{Row: 0, GraphemeCol: 1},
				Items: []CompletionItem{
					{ID: "item-0", InsertText: "X"},
				},
				VisibleIndices: []int{0},
			})

			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

			if got, want := m.buf.Text(), tc.wantText; got != want {
				t.Fatalf("buffer text after accept: got %q, want %q", got, want)
			}
			if got, want := m.CompletionState().Visible, tc.wantPopupVisible; got != want {
				t.Fatalf("popup visibility after accept: got %v, want %v", got, want)
			}
			if got, want := len(completionIntents), 1; got != want {
				t.Fatalf("completion intent count: got %d, want %d", got, want)
			}
			in := completionIntents[0]
			if got, want := in.Kind, IntentCompletionAccept; got != want {
				t.Fatalf("completion intent kind: got %v, want %v", got, want)
			}
			acceptPayload, ok := in.Payload.(CompletionAcceptIntentPayload)
			if !ok {
				t.Fatalf("accept payload type: got %T", in.Payload)
			}
			if got, want := acceptPayload.ItemID, "item-0"; got != want {
				t.Fatalf("accept payload item id: got %q, want %q", got, want)
			}
			if got, want := acceptPayload.InsertText, "X"; got != want {
				t.Fatalf("accept payload insert text: got %q, want %q", got, want)
			}
			if got, want := len(acceptPayload.Edits), 1; got != want {
				t.Fatalf("accept payload edits count: got %d, want %d", got, want)
			}
			if got, want := acceptPayload.Edits[0].Range.Start, (buffer.Pos{Row: 0, GraphemeCol: 1}); got != want {
				t.Fatalf("accept payload edit range start: got %v, want %v", got, want)
			}
			if got, want := acceptPayload.Edits[0].Text, "X"; got != want {
				t.Fatalf("accept payload edit text: got %q, want %q", got, want)
			}

			if tc.wantDocIntentCall {
				if got, want := documentIntentCalls, 1; got != want {
					t.Fatalf("document intent callback calls: got %d, want %d", got, want)
				}
				if got, want := len(documentIntents), 1; got != want {
					t.Fatalf("document intent count: got %d, want %d", got, want)
				}
				if got, want := documentIntents[0].Kind, IntentInsert; got != want {
					t.Fatalf("document intent kind: got %v, want %v", got, want)
				}
				insertPayload, ok := documentIntents[0].Payload.(InsertIntentPayload)
				if !ok {
					t.Fatalf("document insert payload type: got %T", documentIntents[0].Payload)
				}
				if got, want := insertPayload.Text, "X"; got != want {
					t.Fatalf("document insert payload text: got %q, want %q", got, want)
				}
				if got, want := len(insertPayload.Edits), 1; got != want {
					t.Fatalf("document insert payload edits count: got %d, want %d", got, want)
				}
			} else if got, want := documentIntentCalls, 0; got != want {
				t.Fatalf("document intent callback calls: got %d, want %d", got, want)
			}
		})
	}
}

func TestCompletionIntent_MutateDocumentEmitsDualIntentsInOrder(t *testing.T) {
	var order []string
	var completionIntents []CompletionIntent
	var documentIntents []Intent

	m := New(Config{
		Text:                "ab",
		MutationMode:        EmitIntentsOnly,
		CompletionInputMode: CompletionInputMutateDocument,
		OnCompletionIntent: func(batch CompletionIntentBatch) {
			order = append(order, "completion")
			completionIntents = append(completionIntents, batch.Intents...)
		},
		OnIntent: func(batch IntentBatch) IntentDecision {
			order = append(order, "document")
			documentIntents = append(documentIntents, batch.Intents...)
			return IntentDecision{ApplyLocally: true}
		},
	})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Anchor:         buffer.Pos{Row: 0, GraphemeCol: 0},
		Items:          []CompletionItem{{ID: "0", InsertText: "X"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if got, want := order, []string{"completion", "document"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("callback order: got %v, want %v", got, want)
	}
	if got, want := len(completionIntents), 1; got != want {
		t.Fatalf("completion intent count: got %d, want %d", got, want)
	}
	if got, want := completionIntents[0].Kind, IntentCompletionQuery; got != want {
		t.Fatalf("completion intent kind: got %v, want %v", got, want)
	}
	queryPayload, ok := completionIntents[0].Payload.(CompletionQueryIntentPayload)
	if !ok {
		t.Fatalf("completion query payload type: got %T", completionIntents[0].Payload)
	}
	if got, want := queryPayload.Query, "x"; got != want {
		t.Fatalf("completion query payload query: got %q, want %q", got, want)
	}

	if got, want := len(documentIntents), 1; got != want {
		t.Fatalf("document intent count: got %d, want %d", got, want)
	}
	if got, want := documentIntents[0].Kind, IntentInsert; got != want {
		t.Fatalf("document intent kind: got %v, want %v", got, want)
	}
	insertPayload, ok := documentIntents[0].Payload.(InsertIntentPayload)
	if !ok {
		t.Fatalf("document insert payload type: got %T", documentIntents[0].Payload)
	}
	if got, want := insertPayload.Text, "x"; got != want {
		t.Fatalf("document insert payload text: got %q, want %q", got, want)
	}

	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("EmitIntentsOnly should not mutate document: got %q, want %q", got, want)
	}
	if got, want := m.CompletionState().Query, ""; got != want {
		t.Fatalf("query should remain host-controlled when local doc mutation is skipped: got %q, want %q", got, want)
	}
}

func TestCompletionIntent_QueryOnlyUpdatesQueryInEmitIntentsOnly(t *testing.T) {
	var completionIntents []CompletionIntent
	documentIntentCalls := 0

	m := New(Config{
		Text:         "ab",
		MutationMode: EmitIntentsOnly,
		OnCompletionIntent: func(batch CompletionIntentBatch) {
			completionIntents = append(completionIntents, batch.Intents...)
		},
		OnIntent: func(IntentBatch) IntentDecision {
			documentIntentCalls++
			return IntentDecision{ApplyLocally: true}
		},
	})
	m = m.SetCompletionState(CompletionState{
		Visible:        true,
		Items:          []CompletionItem{{ID: "0"}},
		VisibleIndices: []int{0},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if got, want := len(completionIntents), 1; got != want {
		t.Fatalf("completion intent count: got %d, want %d", got, want)
	}
	if got, want := completionIntents[0].Kind, IntentCompletionQuery; got != want {
		t.Fatalf("completion intent kind: got %v, want %v", got, want)
	}
	queryPayload, ok := completionIntents[0].Payload.(CompletionQueryIntentPayload)
	if !ok {
		t.Fatalf("query payload type: got %T", completionIntents[0].Payload)
	}
	if got, want := queryPayload.Query, "x"; got != want {
		t.Fatalf("query payload value: got %q, want %q", got, want)
	}
	if got, want := m.CompletionState().Query, "x"; got != want {
		t.Fatalf("query-only mode should update local query regardless of mutation mode: got %q, want %q", got, want)
	}
	if got, want := documentIntentCalls, 0; got != want {
		t.Fatalf("document intent callback calls: got %d, want %d", got, want)
	}
}
