package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/iw2rmb/flourish/buffer"
)

func TestIntentMode_DefaultModeMutatesAndSkipsOnIntent(t *testing.T) {
	intentCalls := 0
	m := New(Config{
		Text: "ab",
		OnIntent: func(IntentBatch) IntentDecision {
			intentCalls++
			return IntentDecision{ApplyLocally: true}
		},
	})

	m, _ = m.Update(testKeyText("X"))

	if got, want := m.buf.Text(), "Xab"; got != want {
		t.Fatalf("text after insert in default mode: got %q, want %q", got, want)
	}
	if intentCalls != 0 {
		t.Fatalf("OnIntent calls in default mode: got %d, want %d", intentCalls, 0)
	}
}

func TestIntentMode_EmitIntentsOnly_EmitsWithoutLocalApplyOrOnChange(t *testing.T) {
	var intents []Intent
	onChangeCalls := 0

	m := New(Config{
		Text:         "ab",
		MutationMode: EmitIntentsOnly,
		OnIntent: func(batch IntentBatch) IntentDecision {
			intents = append(intents, batch.Intents...)
			return IntentDecision{ApplyLocally: true}
		},
		OnChange: func(buffer.Change) {
			onChangeCalls++
		},
	})

	m, _ = m.Update(testKeyCode(tea.KeyRight))
	m, _ = m.Update(testKeyText("X"))

	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("text in intents-only mode: got %q, want %q", got, want)
	}
	if got, want := m.buf.Cursor(), (buffer.Pos{Row: 0, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor in intents-only mode: got %v, want %v", got, want)
	}
	if got, want := len(intents), 2; got != want {
		t.Fatalf("intent count in intents-only mode: got %d, want %d", got, want)
	}
	if got, want := onChangeCalls, 0; got != want {
		t.Fatalf("OnChange calls in intents-only mode: got %d, want %d", got, want)
	}
}

func TestIntentMode_EmitIntentsAndMutate_DecisionFalseSkipsApplyAndOnChange(t *testing.T) {
	var intents []Intent
	onChangeCalls := 0

	m := New(Config{
		Text:         "ab",
		MutationMode: EmitIntentsAndMutate,
		OnIntent: func(batch IntentBatch) IntentDecision {
			intents = append(intents, batch.Intents...)
			return IntentDecision{ApplyLocally: false}
		},
		OnChange: func(buffer.Change) {
			onChangeCalls++
		},
	})

	m, _ = m.Update(testKeyText("X"))

	if got, want := len(intents), 1; got != want {
		t.Fatalf("intent count with ApplyLocally=false: got %d, want %d", got, want)
	}
	if got, want := m.buf.Text(), "ab"; got != want {
		t.Fatalf("text with ApplyLocally=false: got %q, want %q", got, want)
	}
	if got, want := onChangeCalls, 0; got != want {
		t.Fatalf("OnChange calls with ApplyLocally=false: got %d, want %d", got, want)
	}
}

func TestIntentMode_ParityMutateVsEmitAndMutateApplyTrue(t *testing.T) {
	a := New(Config{
		Text: "hello",
	})
	var gotBatches []IntentBatch
	b := New(Config{
		Text:         "hello",
		MutationMode: EmitIntentsAndMutate,
		OnIntent: func(batch IntentBatch) IntentDecision {
			gotBatches = append(gotBatches, batch)
			return IntentDecision{ApplyLocally: true}
		},
	})

	seq := []tea.KeyPressMsg{
		testKeyCode(tea.KeyRight),
		testKeyCode(tea.KeyRight),
		testKeyCode(tea.KeyRight, tea.ModShift),
		testKeyText("Z"),
		testKeyCode(tea.KeyRight),
		testKeyCode(tea.KeyBackspace),
		testKeyCode('z', tea.ModCtrl),
		testKeyCode('y', tea.ModCtrl),
	}

	for _, msg := range seq {
		a, _ = a.Update(msg)
		b, _ = b.Update(msg)
	}

	if got, want := b.buf.Text(), a.buf.Text(); got != want {
		t.Fatalf("text parity mismatch: got %q, want %q", got, want)
	}
	if got, want := b.buf.Cursor(), a.buf.Cursor(); got != want {
		t.Fatalf("cursor parity mismatch: got %v, want %v", got, want)
	}
	gotSel, gotSelOK := b.buf.Selection()
	wantSel, wantSelOK := a.buf.Selection()
	if gotSelOK != wantSelOK || gotSel != wantSel {
		t.Fatalf("selection parity mismatch: got (%v,%v), want (%v,%v)", gotSel, gotSelOK, wantSel, wantSelOK)
	}
	if len(gotBatches) == 0 {
		t.Fatalf("expected intent batches in EmitIntentsAndMutate mode")
	}
}

func TestIntentEmission_KeyKindsAndSelectionAwareDelete(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		setup   func(*Model)
		msg     tea.KeyPressMsg
		want    IntentKind
		checkFn func(t *testing.T, in Intent)
	}{
		{
			name: "insert",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			msg:  testKeyText("X"),
			want: IntentInsert,
			checkFn: func(t *testing.T, in Intent) {
				t.Helper()
				p, ok := in.Payload.(InsertIntentPayload)
				if !ok {
					t.Fatalf("insert payload type: got %T", in.Payload)
				}
				if got, want := p.Text, "X"; got != want {
					t.Fatalf("insert payload text: got %q, want %q", got, want)
				}
			},
		},
		{
			name: "delete selection",
			cfg:  Config{Text: "hello", MutationMode: EmitIntentsOnly},
			setup: func(m *Model) {
				m.buf.SetSelection(buffer.Range{
					Start: buffer.Pos{Row: 0, GraphemeCol: 1},
					End:   buffer.Pos{Row: 0, GraphemeCol: 4},
				})
			},
			msg:  testKeyCode(tea.KeyBackspace),
			want: IntentDelete,
			checkFn: func(t *testing.T, in Intent) {
				t.Helper()
				p, ok := in.Payload.(DeleteIntentPayload)
				if !ok {
					t.Fatalf("delete payload type: got %T", in.Payload)
				}
				if got, want := p.Direction, DeleteSelection; got != want {
					t.Fatalf("delete direction: got %v, want %v", got, want)
				}
			},
		},
		{
			name: "move",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			msg:  testKeyCode(tea.KeyRight),
			want: IntentMove,
		},
		{
			name: "move page down",
			cfg:  Config{Text: "0\n1\n2\n3\n4\n5", MutationMode: EmitIntentsOnly},
			setup: func(m *Model) {
				*m = m.SetSize(10, 3)
			},
			msg:  testKeyCode(tea.KeyPgDown),
			want: IntentMove,
			checkFn: func(t *testing.T, in Intent) {
				t.Helper()
				p, ok := in.Payload.(MoveIntentPayload)
				if !ok {
					t.Fatalf("move payload type: got %T", in.Payload)
				}
				if got, want := p.Move, (buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirDown, Count: 3}); got != want {
					t.Fatalf("move payload: got %+v, want %+v", got, want)
				}
			},
		},
		{
			name: "select",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			msg:  testKeyCode(tea.KeyRight, tea.ModShift),
			want: IntentSelect,
		},
		{
			name: "undo",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			setup: func(m *Model) {
				m.buf.InsertText("X")
			},
			msg:  testKeyCode('z', tea.ModCtrl),
			want: IntentUndo,
		},
		{
			name: "redo",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			setup: func(m *Model) {
				m.buf.InsertText("X")
				_ = m.buf.Undo()
			},
			msg:  testKeyCode('y', tea.ModCtrl),
			want: IntentRedo,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got IntentBatch
			tc.cfg.OnIntent = func(batch IntentBatch) IntentDecision {
				got = batch
				return IntentDecision{ApplyLocally: false}
			}
			m := New(tc.cfg)
			if tc.setup != nil {
				tc.setup(&m)
			}

			m, _ = m.Update(tc.msg)

			if gotN := len(got.Intents); gotN != 1 {
				t.Fatalf("intent count: got %d, want %d", gotN, 1)
			}
			intent := got.Intents[0]
			if intent.Kind != tc.want {
				t.Fatalf("intent kind: got %v, want %v", intent.Kind, tc.want)
			}
			if tc.checkFn != nil {
				tc.checkFn(t, intent)
			}
		})
	}
}

func TestIntentMode_ReadOnlyBlocksMutationIntents(t *testing.T) {
	var intents []Intent
	m := New(Config{
		Text:         "ab",
		ReadOnly:     true,
		MutationMode: EmitIntentsOnly,
		OnIntent: func(batch IntentBatch) IntentDecision {
			intents = append(intents, batch.Intents...)
			return IntentDecision{ApplyLocally: true}
		},
	})

	m, _ = m.Update(testKeyText("X"))
	m, _ = m.Update(testKeyCode(tea.KeyBackspace))
	m, _ = m.Update(testKeyCode('z', tea.ModCtrl))
	m, _ = m.Update(testKeyCode(tea.KeyRight))
	m, _ = m.Update(testKeyCode(tea.KeyRight, tea.ModShift))

	if got, want := len(intents), 2; got != want {
		t.Fatalf("intent count in read-only mode: got %d, want %d", got, want)
	}
	if got, want := intents[0].Kind, IntentMove; got != want {
		t.Fatalf("first intent kind in read-only mode: got %v, want %v", got, want)
	}
	if got, want := intents[1].Kind, IntentSelect; got != want {
		t.Fatalf("second intent kind in read-only mode: got %v, want %v", got, want)
	}
}

func TestIntentEmission_UndoRedoNoHistoryEmitNothing(t *testing.T) {
	intents := make([]Intent, 0, 2)
	m := New(Config{
		Text:         "ab",
		MutationMode: EmitIntentsOnly,
		OnIntent: func(batch IntentBatch) IntentDecision {
			intents = append(intents, batch.Intents...)
			return IntentDecision{ApplyLocally: false}
		},
	})

	m, _ = m.Update(testKeyCode('z', tea.ModCtrl))
	m, _ = m.Update(testKeyCode('y', tea.ModCtrl))

	if got, want := len(intents), 0; got != want {
		t.Fatalf("intent count with empty undo/redo history: got %d, want %d", got, want)
	}
}
