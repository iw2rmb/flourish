package editor

import "github.com/iw2rmb/flourish/buffer"

// MutationMode controls whether key handling mutates the local buffer,
// emits intents to the host, or both.
type MutationMode uint8

const (
	// MutateInEditor keeps current behavior: key handling mutates editor state.
	MutateInEditor MutationMode = iota
	// EmitIntentsOnly emits intents and does not apply local mutations.
	EmitIntentsOnly
	// EmitIntentsAndMutate emits intents and applies local mutations when the
	// host decision allows it.
	EmitIntentsAndMutate
)

// IntentKind identifies the semantic action requested by input handling.
type IntentKind uint8

const (
	IntentInsert IntentKind = iota
	IntentDelete
	IntentMove
	IntentSelect
	IntentUndo
	IntentRedo
	IntentPaste
)

// EditorState captures buffer-local state before an intent is executed.
type EditorState struct {
	Version   uint64
	Cursor    buffer.Pos
	Selection buffer.SelectionState
}

// Intent is a typed semantic action emitted from key processing.
type Intent struct {
	Kind    IntentKind
	Before  EditorState
	Payload any
}

// IntentBatch groups intents produced from one input event.
type IntentBatch struct {
	Intents []Intent
}

// IntentDecision controls whether the editor applies mutations locally.
// It is used in EmitIntentsAndMutate mode.
type IntentDecision struct {
	ApplyLocally bool
}

// DeleteDirection identifies requested delete direction semantics.
type DeleteDirection uint8

const (
	DeleteBackward DeleteDirection = iota
	DeleteForward
	DeleteSelection
)

// InsertIntentPayload describes an insert action.
type InsertIntentPayload struct {
	Text string
	// Edits carries deterministic apply edits for complex insertion sources
	// (for example, ghost accepts). For regular typing, this is empty.
	Edits []buffer.TextEdit
}

// DeleteIntentPayload describes a delete action.
type DeleteIntentPayload struct {
	Direction DeleteDirection
}

// MoveIntentPayload describes a cursor move action.
type MoveIntentPayload struct {
	Move buffer.Move
}

// SelectIntentPayload describes a selection-extending move action.
type SelectIntentPayload struct {
	Move buffer.Move
}

// UndoIntentPayload marks an undo request.
type UndoIntentPayload struct{}

// RedoIntentPayload marks a redo request.
type RedoIntentPayload struct{}

// PasteIntentPayload describes a paste request.
type PasteIntentPayload struct {
	Text string
}

func editorStateFromBuffer(b *buffer.Buffer) EditorState {
	if b == nil {
		return EditorState{}
	}
	sel := buffer.SelectionState{}
	if r, ok := b.Selection(); ok {
		sel = buffer.SelectionState{Active: true, Range: r}
	}
	return EditorState{
		Version:   b.Version(),
		Cursor:    b.Cursor(),
		Selection: sel,
	}
}

func normalizeMutationMode(mode MutationMode) MutationMode {
	switch mode {
	case MutateInEditor, EmitIntentsOnly, EmitIntentsAndMutate:
		return mode
	default:
		return MutateInEditor
	}
}

func cloneTextEdits(in []buffer.TextEdit) []buffer.TextEdit {
	if len(in) == 0 {
		return nil
	}
	out := make([]buffer.TextEdit, len(in))
	copy(out, in)
	return out
}
