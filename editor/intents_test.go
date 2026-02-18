package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

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
		OnChange: func(ChangeEvent) {
			onChangeCalls++
		},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

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
		OnChange: func(ChangeEvent) {
			onChangeCalls++
		},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

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
	cbA := &memClipboard{t: t, s: "Z"}
	cbB := &memClipboard{t: t, s: "Z"}

	a := New(Config{
		Text:      "hello",
		Clipboard: cbA,
	})
	var gotBatches []IntentBatch
	b := New(Config{
		Text:         "hello",
		Clipboard:    cbB,
		MutationMode: EmitIntentsAndMutate,
		OnIntent: func(batch IntentBatch) IntentDecision {
			gotBatches = append(gotBatches, batch)
			return IntentDecision{ApplyLocally: true}
		},
	})

	seq := []tea.KeyMsg{
		{Type: tea.KeyRight},
		{Type: tea.KeyRight},
		{Type: tea.KeyShiftRight},
		{Type: tea.KeyCtrlV},
		{Type: tea.KeyRight},
		{Type: tea.KeyBackspace},
		{Type: tea.KeyCtrlZ},
		{Type: tea.KeyCtrlY},
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

func TestIntentEmission_KeyKindsAndSelectionAwareDeleteAndPaste(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		setup   func(*Model)
		msg     tea.KeyMsg
		want    IntentKind
		checkFn func(t *testing.T, in Intent)
	}{
		{
			name: "insert",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")},
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
			msg:  tea.KeyMsg{Type: tea.KeyBackspace},
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
			msg:  tea.KeyMsg{Type: tea.KeyRight},
			want: IntentMove,
		},
		{
			name: "select",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			msg:  tea.KeyMsg{Type: tea.KeyShiftRight},
			want: IntentSelect,
		},
		{
			name: "undo",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			msg:  tea.KeyMsg{Type: tea.KeyCtrlZ},
			want: IntentUndo,
		},
		{
			name: "redo",
			cfg:  Config{Text: "ab", MutationMode: EmitIntentsOnly},
			msg:  tea.KeyMsg{Type: tea.KeyCtrlY},
			want: IntentRedo,
		},
		{
			name: "paste",
			cfg: Config{
				Text:         "ab",
				Clipboard:    &memClipboard{t: t, s: "X\r\nY\rZ"},
				MutationMode: EmitIntentsOnly,
			},
			msg:  tea.KeyMsg{Type: tea.KeyCtrlV},
			want: IntentPaste,
			checkFn: func(t *testing.T, in Intent) {
				t.Helper()
				p, ok := in.Payload.(PasteIntentPayload)
				if !ok {
					t.Fatalf("paste payload type: got %T", in.Payload)
				}
				if got, want := p.Text, "X\nY\nZ"; got != want {
					t.Fatalf("paste payload text: got %q, want %q", got, want)
				}
			},
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
		Clipboard:    &memClipboard{t: t, s: "P"},
		MutationMode: EmitIntentsOnly,
		OnIntent: func(batch IntentBatch) IntentDecision {
			intents = append(intents, batch.Intents...)
			return IntentDecision{ApplyLocally: true}
		},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftRight})

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
