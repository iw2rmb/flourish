package editor

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/iw2rmb/flourish/buffer"
)

type localMutationOp func(*Model)

type completionKeyResult struct {
	completionBatch     CompletionIntentBatch
	documentBatch       IntentBatch
	completionMutations []localMutationOp
	documentMutations   []localMutationOp
}

func (m Model) updateKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if !m.focused || m.buf == nil {
		return m, nil
	}

	before := editorStateFromBuffer(m.buf)
	if completion, handled := (&m).buildCompletionIntentsFromKey(msg, before); handled {
		if len(completion.completionBatch.Intents) > 0 && m.cfg.OnCompletionIntent != nil {
			m.cfg.OnCompletionIntent(completion.completionBatch)
		}
		applyDocumentLocally := (&m).emitDocumentIntentsAndResolveApply(completion.documentBatch)
		for _, op := range completion.completionMutations {
			op(&m)
		}
		if applyDocumentLocally {
			for _, op := range completion.documentMutations {
				op(&m)
			}
		}
		return m, nil
	}

	batch, mutations := (&m).buildIntentsFromKey(msg, before)
	applyLocally := (&m).emitDocumentIntentsAndResolveApply(batch)
	if applyLocally {
		for _, op := range mutations {
			op(&m)
		}
	}

	return m, nil
}

func (m *Model) emitDocumentIntentsAndResolveApply(batch IntentBatch) bool {
	mode := normalizeMutationMode(m.cfg.MutationMode)

	switch mode {
	case MutateInEditor:
		return true
	case EmitIntentsOnly:
		if len(batch.Intents) > 0 && m.cfg.OnIntent != nil {
			m.cfg.OnIntent(batch)
		}
		return false
	case EmitIntentsAndMutate:
		if len(batch.Intents) == 0 || m.cfg.OnIntent == nil {
			return true
		}
		decision := m.cfg.OnIntent(batch)
		return decision.ApplyLocally
	default:
		return true
	}
}

func (m *Model) buildIntentsFromKey(msg tea.KeyPressMsg, before EditorState) (IntentBatch, []localMutationOp) {
	batch := IntentBatch{}
	mutations := make([]localMutationOp, 0, 1)
	appendIntent := func(kind IntentKind, payload any) {
		batch.Intents = append(batch.Intents, Intent{Kind: kind, Before: before, Payload: payload})
	}

	km := m.cfg.KeyMap
	ga := normalizeGhostAccept(m.cfg.GhostAccept)

	if m.cfg.GhostProvider != nil && !m.cfg.ReadOnly {
		if ga.AcceptTab && isTabKey(msg) {
			if ghost, ok := m.ghostForCursor(); ok && len(ghost.Edits) > 0 {
				edits := cloneTextEdits(ghost.Edits)
				appendIntent(IntentInsert, InsertIntentPayload{Text: ghost.Text, Edits: edits})
				mutations = append(mutations, func(mm *Model) {
					mm.buf.Apply(edits...)
				})
				return batch, mutations
			}
		}
		if ga.AcceptRight && key.Matches(msg, km.Right) {
			if ghost, ok := m.ghostForCursor(); ok && len(ghost.Edits) > 0 {
				edits := cloneTextEdits(ghost.Edits)
				appendIntent(IntentInsert, InsertIntentPayload{Text: ghost.Text, Edits: edits})
				mutations = append(mutations, func(mm *Model) {
					mm.buf.Apply(edits...)
				})
				return batch, mutations
			}
		}
	}

	appendMove := func(move buffer.Move) {
		kind := IntentMove
		var payload any = MoveIntentPayload{Move: move}
		if move.Extend {
			kind = IntentSelect
			payload = SelectIntentPayload{Move: move}
		}
		appendIntent(kind, payload)
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	}

	switch {
	case key.Matches(msg, km.Left):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirLeft})
	case key.Matches(msg, km.Right):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirRight})
	case key.Matches(msg, km.Up):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirUp})
	case key.Matches(msg, km.Down):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirDown})
	case key.Matches(msg, km.PageUp):
		appendMove(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirUp, Count: m.pageMoveCount()})
	case key.Matches(msg, km.PageDown):
		appendMove(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirDown, Count: m.pageMoveCount()})

	case key.Matches(msg, km.ShiftLeft):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirLeft, Extend: true})
	case key.Matches(msg, km.ShiftRight):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirRight, Extend: true})
	case key.Matches(msg, km.ShiftUp):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirUp, Extend: true})
	case key.Matches(msg, km.ShiftDown):
		appendMove(buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirDown, Extend: true})

	case key.Matches(msg, km.WordLeft):
		appendMove(buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirLeft})
	case key.Matches(msg, km.WordRight):
		appendMove(buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirRight})
	case key.Matches(msg, km.WordShiftLeft):
		appendMove(buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirLeft, Extend: true})
	case key.Matches(msg, km.WordShiftRight):
		appendMove(buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirRight, Extend: true})

	case key.Matches(msg, km.Home):
		appendMove(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirHome})
	case key.Matches(msg, km.End):
		appendMove(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirEnd})

	case key.Matches(msg, km.Backspace):
		if !m.cfg.ReadOnly {
			dir := DeleteBackward
			if _, ok := m.buf.Selection(); ok {
				dir = DeleteSelection
			}
			appendIntent(IntentDelete, DeleteIntentPayload{Direction: dir})
			mutations = append(mutations, func(mm *Model) { mm.buf.DeleteBackward() })
		}
	case key.Matches(msg, km.Delete):
		if !m.cfg.ReadOnly {
			dir := DeleteForward
			if _, ok := m.buf.Selection(); ok {
				dir = DeleteSelection
			}
			appendIntent(IntentDelete, DeleteIntentPayload{Direction: dir})
			mutations = append(mutations, func(mm *Model) { mm.buf.DeleteForward() })
		}
	case key.Matches(msg, km.Enter):
		if !m.cfg.ReadOnly {
			appendIntent(IntentInsert, InsertIntentPayload{Text: "\n"})
			mutations = append(mutations, func(mm *Model) { mm.buf.InsertNewline() })
		}

	case key.Matches(msg, km.Undo):
		if !m.cfg.ReadOnly {
			if m.buf.CanUndo() {
				appendIntent(IntentUndo, UndoIntentPayload{})
				mutations = append(mutations, func(mm *Model) { _ = mm.buf.Undo() })
			}
		}
	case key.Matches(msg, km.Redo):
		if !m.cfg.ReadOnly {
			if m.buf.CanRedo() {
				appendIntent(IntentRedo, RedoIntentPayload{})
				mutations = append(mutations, func(mm *Model) { _ = mm.buf.Redo() })
			}
		}

	default:
		if isTabKey(msg) {
			if !m.cfg.ReadOnly {
				appendIntent(IntentInsert, InsertIntentPayload{Text: "\t"})
				mutations = append(mutations, func(mm *Model) { mm.buf.InsertGrapheme("\t") })
			}
			return batch, mutations
		}
		if isSpaceKey(msg) && !hasAltMod(msg) {
			if !m.cfg.ReadOnly {
				appendIntent(IntentInsert, InsertIntentPayload{Text: " "})
				mutations = append(mutations, func(mm *Model) { mm.buf.InsertGrapheme(" ") })
			}
			return batch, mutations
		}

		text := keyText(msg)
		if text != "" && !hasAltMod(msg) {
			if !m.cfg.ReadOnly {
				appendIntent(IntentInsert, InsertIntentPayload{Text: text})
				mutations = append(mutations, func(mm *Model) { mm.buf.InsertText(text) })
			}
		}
	}

	return batch, mutations
}

func (m *Model) pageMoveCount() int {
	count := m.visibleRowCount()
	if count <= 0 {
		return 1
	}
	return count
}

func isTabKey(msg tea.KeyPressMsg) bool {
	return msg.Key().Code == tea.KeyTab
}

func isSpaceKey(msg tea.KeyPressMsg) bool {
	k := msg.Key()
	return k.Code == tea.KeySpace || k.Text == " "
}

func hasAltMod(msg tea.KeyPressMsg) bool {
	return msg.Key().Mod&tea.ModAlt != 0
}

func keyText(msg tea.KeyPressMsg) string {
	return msg.Key().Text
}
