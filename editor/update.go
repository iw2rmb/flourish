package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

type localMutationOp func(*Model)

type completionKeyResult struct {
	completionBatch     CompletionIntentBatch
	documentBatch       IntentBatch
	completionMutations []localMutationOp
	documentMutations   []localMutationOp
}

func (m Model) updateKey(msg tea.KeyMsg) (Model, tea.Cmd) {
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

func (m *Model) buildIntentsFromKey(msg tea.KeyMsg, before EditorState) (IntentBatch, []localMutationOp) {
	batch := IntentBatch{}
	mutations := make([]localMutationOp, 0, 1)
	appendIntent := func(kind IntentKind, payload any) {
		batch.Intents = append(batch.Intents, Intent{Kind: kind, Before: before, Payload: payload})
	}

	// Paste events should always insert literal text and never trigger shortcuts.
	if msg.Type == tea.KeyRunes && msg.Paste && len(msg.Runes) > 0 {
		if !m.cfg.ReadOnly {
			text := string(msg.Runes)
			appendIntent(IntentPaste, PasteIntentPayload{Text: text})
			mutations = append(mutations, func(mm *Model) {
				mm.buf.InsertText(text)
			})
		}
		return batch, mutations
	}

	km := m.cfg.KeyMap
	ga := normalizeGhostAccept(m.cfg.GhostAccept)

	if m.cfg.GhostProvider != nil && !m.cfg.ReadOnly {
		if ga.AcceptTab && msg.Type == tea.KeyTab {
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

	case key.Matches(msg, km.Copy):
		m.copySelection()
	case key.Matches(msg, km.Cut):
		if m.cfg.ReadOnly {
			m.copySelection()
			return batch, mutations
		}
		m.copySelection()
		if _, ok := m.buf.Selection(); ok {
			appendIntent(IntentDelete, DeleteIntentPayload{Direction: DeleteSelection})
			mutations = append(mutations, func(mm *Model) { mm.buf.DeleteSelection() })
		}
	case key.Matches(msg, km.Paste):
		if !m.cfg.ReadOnly {
			if text, ok := m.readClipboardText(); ok {
				appendIntent(IntentPaste, PasteIntentPayload{Text: text})
				mutations = append(mutations, func(mm *Model) { mm.buf.InsertText(text) })
			}
		}

	default:
		if msg.Type == tea.KeyTab {
			if !m.cfg.ReadOnly {
				appendIntent(IntentInsert, InsertIntentPayload{Text: "\t"})
				mutations = append(mutations, func(mm *Model) { mm.buf.InsertGrapheme("\t") })
			}
			return batch, mutations
		}
		if msg.Type == tea.KeySpace && !msg.Alt {
			if !m.cfg.ReadOnly {
				appendIntent(IntentInsert, InsertIntentPayload{Text: " "})
				mutations = append(mutations, func(mm *Model) { mm.buf.InsertGrapheme(" ") })
			}
			return batch, mutations
		}

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && !msg.Alt {
			if !m.cfg.ReadOnly {
				text := string(msg.Runes)
				appendIntent(IntentInsert, InsertIntentPayload{Text: text})
				mutations = append(mutations, func(mm *Model) { mm.buf.InsertText(text) })
			}
		}
	}

	return batch, mutations
}

func (m Model) copySelection() {
	if m.cfg.Clipboard == nil || m.buf == nil {
		return
	}
	r, ok := m.buf.Selection()
	if !ok {
		return
	}
	s := m.buf.TextInRange(r)
	if s == "" {
		return
	}
	_ = m.cfg.Clipboard.WriteText(s)
}

func (m Model) readClipboardText() (string, bool) {
	if m.cfg.Clipboard == nil || m.buf == nil {
		return "", false
	}
	s, err := m.cfg.Clipboard.ReadText()
	if err != nil || s == "" {
		return "", false
	}
	// Normalize newlines from external sources.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s, true
}

func (m *Model) pageMoveCount() int {
	count := m.visibleRowCount()
	if count <= 0 {
		return 1
	}
	return count
}
