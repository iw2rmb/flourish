package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
	"github.com/iw2rmb/flourish/internal/grapheme"
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

	switch {
	case key.Matches(msg, km.Left):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirLeft}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.Right):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirRight}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.Up):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirUp}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.Down):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirDown}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })

	case key.Matches(msg, km.ShiftLeft):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirLeft, Extend: true}
		appendIntent(IntentSelect, SelectIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.ShiftRight):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirRight, Extend: true}
		appendIntent(IntentSelect, SelectIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.ShiftUp):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirUp, Extend: true}
		appendIntent(IntentSelect, SelectIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.ShiftDown):
		move := buffer.Move{Unit: buffer.MoveGrapheme, Dir: buffer.DirDown, Extend: true}
		appendIntent(IntentSelect, SelectIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })

	case key.Matches(msg, km.WordLeft):
		move := buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirLeft}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.WordRight):
		move := buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirRight}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.WordShiftLeft):
		move := buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirLeft, Extend: true}
		appendIntent(IntentSelect, SelectIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.WordShiftRight):
		move := buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirRight, Extend: true}
		appendIntent(IntentSelect, SelectIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })

	case key.Matches(msg, km.Home):
		move := buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirHome}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })
	case key.Matches(msg, km.End):
		move := buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirEnd}
		appendIntent(IntentMove, MoveIntentPayload{Move: move})
		mutations = append(mutations, func(mm *Model) { mm.buf.Move(move) })

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
			appendIntent(IntentUndo, UndoIntentPayload{})
			mutations = append(mutations, func(mm *Model) { _ = mm.buf.Undo() })
		}
	case key.Matches(msg, km.Redo):
		if !m.cfg.ReadOnly {
			appendIntent(IntentRedo, RedoIntentPayload{})
			mutations = append(mutations, func(mm *Model) { _ = mm.buf.Redo() })
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
	s := textInRange(m.buf.Text(), r)
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

func textInRange(text string, r buffer.Range) string {
	r = buffer.NormalizeRange(r)
	if r.IsEmpty() {
		return ""
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}

	if r.Start.Row < 0 || r.Start.Row >= len(lines) || r.End.Row < 0 || r.End.Row >= len(lines) {
		return ""
	}

	if r.Start.Row == r.End.Row {
		rr := grapheme.Split(lines[r.Start.Row])
		if r.Start.GraphemeCol < 0 || r.End.GraphemeCol < 0 || r.Start.GraphemeCol > len(rr) || r.End.GraphemeCol > len(rr) {
			return ""
		}
		return grapheme.Join(rr[r.Start.GraphemeCol:r.End.GraphemeCol])
	}

	var sb strings.Builder
	for row := r.Start.Row; row <= r.End.Row; row++ {
		if row > r.Start.Row {
			sb.WriteByte('\n')
		}
		rr := grapheme.Split(lines[row])
		startCol := 0
		endCol := len(rr)
		if row == r.Start.Row {
			startCol = r.Start.GraphemeCol
		}
		if row == r.End.Row {
			endCol = r.End.GraphemeCol
		}
		if startCol < 0 || endCol < 0 || startCol > len(rr) || endCol > len(rr) || startCol > endCol {
			return ""
		}
		sb.WriteString(grapheme.Join(rr[startCol:endCol]))
	}
	return sb.String()
}

func (m *Model) buildCompletionIntentsFromKey(msg tea.KeyMsg, before EditorState) (completionKeyResult, bool) {
	ckm := m.cfg.CompletionKeyMap
	km := m.cfg.KeyMap
	result := completionKeyResult{
		completionMutations: make([]localMutationOp, 0, 2),
		documentMutations:   make([]localMutationOp, 0, 2),
	}
	appendCompletionIntent := func(kind CompletionIntentKind, payload any) {
		result.completionBatch.Intents = append(result.completionBatch.Intents, CompletionIntent{
			Kind:    kind,
			Before:  before,
			Payload: payload,
		})
	}
	appendDocumentIntent := func(kind IntentKind, payload any) {
		result.documentBatch.Intents = append(result.documentBatch.Intents, Intent{
			Kind:    kind,
			Before:  before,
			Payload: payload,
		})
	}
	appendNavigate := func(delta int) {
		payload := m.completionNavigatePayload(delta)
		appendCompletionIntent(IntentCompletionNavigate, payload)
		result.completionMutations = append(result.completionMutations, func(mm *Model) {
			mm.moveCompletionSelection(delta)
		})
	}

	if key.Matches(msg, ckm.Trigger) {
		appendCompletionIntent(IntentCompletionTrigger, CompletionTriggerIntentPayload{Anchor: before.Cursor})
		result.completionMutations = append(result.completionMutations, func(mm *Model) {
			mm.openCompletionAtCursor()
		})
		return result, true
	}

	if !m.completionState.Visible {
		return result, false
	}

	switch {
	case key.Matches(msg, ckm.Dismiss):
		appendCompletionIntent(IntentCompletionDismiss, CompletionDismissIntentPayload{})
		result.completionMutations = append(result.completionMutations, func(mm *Model) {
			*mm = mm.ClearCompletion()
		})
		return result, true
	case key.Matches(msg, ckm.Next):
		appendNavigate(1)
		return result, true
	case key.Matches(msg, ckm.Prev):
		appendNavigate(-1)
		return result, true
	case key.Matches(msg, ckm.PageNext):
		step := m.cfg.CompletionMaxVisibleRows
		if step <= 0 {
			step = defaultCompletionMaxVisibleRows
		}
		appendNavigate(step)
		return result, true
	case key.Matches(msg, ckm.PagePrev):
		step := m.cfg.CompletionMaxVisibleRows
		if step <= 0 {
			step = defaultCompletionMaxVisibleRows
		}
		appendNavigate(-step)
		return result, true
	}

	if key.Matches(msg, ckm.Accept) || (msg.Type == tea.KeyTab && ckm.AcceptTab) {
		acceptPayload, ok := m.acceptCompletionPayload()
		if !ok {
			return result, true
		}
		completionPayload := acceptPayload
		completionPayload.Edits = cloneTextEdits(completionPayload.Edits)
		appendCompletionIntent(IntentCompletionAccept, completionPayload)
		if !m.cfg.ReadOnly {
			documentEdits := cloneTextEdits(acceptPayload.Edits)
			appendDocumentIntent(IntentInsert, InsertIntentPayload{Text: acceptPayload.InsertText, Edits: cloneTextEdits(documentEdits)})
			result.documentMutations = append(result.documentMutations, func(mm *Model) {
				mm.buf.Apply(documentEdits...)
				*mm = mm.ClearCompletion()
			})
		}
		return result, true
	}

	mode := normalizeCompletionInputMode(m.cfg.CompletionInputMode)
	if m.cfg.ReadOnly {
		mode = CompletionInputQueryOnly
	}

	if mode == CompletionInputQueryOnly {
		if query, ok := m.nextCompletionQueryFromKey(msg); ok {
			appendCompletionIntent(IntentCompletionQuery, CompletionQueryIntentPayload{Query: query})
			result.completionMutations = append(result.completionMutations, func(mm *Model) {
				mm.setCompletionQuery(query)
			})
			return result, true
		}
		return result, false
	}

	if key.Matches(msg, km.Backspace) {
		query, ok := m.nextCompletionQueryForMutateDocumentKey(msg)
		if !ok {
			query = m.completionState.Query
		}
		appendCompletionIntent(IntentCompletionQuery, CompletionQueryIntentPayload{Query: query})
		dir := DeleteBackward
		if _, ok := m.buf.Selection(); ok {
			dir = DeleteSelection
		}
		appendDocumentIntent(IntentDelete, DeleteIntentPayload{Direction: dir})
		result.documentMutations = append(result.documentMutations, func(mm *Model) {
			mm.buf.DeleteBackward()
			mm.recomputeCompletionQueryFromAnchor()
		})
		return result, true
	}
	if msg.Type == tea.KeySpace && !msg.Alt {
		query, ok := m.nextCompletionQueryForMutateDocumentKey(msg)
		if !ok {
			query = m.completionState.Query
		}
		appendCompletionIntent(IntentCompletionQuery, CompletionQueryIntentPayload{Query: query})
		appendDocumentIntent(IntentInsert, InsertIntentPayload{Text: " "})
		result.documentMutations = append(result.documentMutations, func(mm *Model) {
			mm.buf.InsertGrapheme(" ")
			mm.recomputeCompletionQueryFromAnchor()
		})
		return result, true
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && !msg.Alt && !msg.Paste {
		text := string(msg.Runes)
		query, ok := m.nextCompletionQueryForMutateDocumentKey(msg)
		if !ok {
			query = m.completionState.Query
		}
		appendCompletionIntent(IntentCompletionQuery, CompletionQueryIntentPayload{Query: query})
		appendDocumentIntent(IntentInsert, InsertIntentPayload{Text: text})
		result.documentMutations = append(result.documentMutations, func(mm *Model) {
			mm.buf.InsertText(text)
			mm.recomputeCompletionQueryFromAnchor()
		})
		return result, true
	}

	return result, false
}

func (m *Model) openCompletionAtCursor() {
	state := m.completionState
	state.Visible = true
	state.Query = ""
	if m.buf != nil {
		state.Anchor = m.buf.Cursor()
	}
	m.recomputeCompletionFilter(&state)
	m.completionState = state
}

func (m *Model) moveCompletionSelection(delta int) {
	state := m.completionState
	m.normalizeCompletionRuntimeState(&state)
	if len(state.VisibleIndices) == 0 {
		m.completionState = state
		return
	}
	state.Selected = clampInt(state.Selected+delta, 0, len(state.VisibleIndices)-1)
	m.completionState = state
}

func (m *Model) completionNavigatePayload(delta int) CompletionNavigateIntentPayload {
	state := m.completionState
	m.normalizeCompletionRuntimeState(&state)

	payload := CompletionNavigateIntentPayload{
		Delta:     delta,
		Selected:  0,
		ItemIndex: -1,
	}
	if len(state.VisibleIndices) == 0 {
		return payload
	}

	selected := clampInt(state.Selected+delta, 0, len(state.VisibleIndices)-1)
	payload.Selected = selected
	payload.ItemIndex = state.VisibleIndices[selected]
	return payload
}

func (m *Model) acceptCompletionPayload() (CompletionAcceptIntentPayload, bool) {
	state := m.completionState
	m.normalizeCompletionRuntimeState(&state)
	if len(state.VisibleIndices) == 0 || len(state.Items) == 0 {
		return CompletionAcceptIntentPayload{}, false
	}

	selected := clampInt(state.Selected, 0, len(state.VisibleIndices)-1)
	itemIdx := state.VisibleIndices[selected]
	if itemIdx < 0 || itemIdx >= len(state.Items) {
		return CompletionAcceptIntentPayload{}, false
	}

	item := state.Items[itemIdx]
	edits := cloneTextEdits(item.Edits)
	if len(edits) == 0 {
		edits = []buffer.TextEdit{{
			Range: buffer.Range{
				Start: state.Anchor,
				End:   state.Anchor,
			},
			Text: item.InsertText,
		}}
	}

	return CompletionAcceptIntentPayload{
		ItemID:       item.ID,
		ItemIndex:    itemIdx,
		VisibleIndex: selected,
		InsertText:   item.InsertText,
		Edits:        edits,
	}, true
}

func (m *Model) nextCompletionQueryFromKey(msg tea.KeyMsg) (string, bool) {
	if key.Matches(msg, m.cfg.KeyMap.Backspace) {
		parts := grapheme.Split(m.completionState.Query)
		if len(parts) == 0 {
			return "", true
		}
		return grapheme.Join(parts[:len(parts)-1]), true
	}
	if msg.Type == tea.KeySpace && !msg.Alt {
		return m.completionState.Query + " ", true
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && !msg.Alt {
		return m.completionState.Query + string(msg.Runes), true
	}
	return "", false
}

func (m *Model) nextCompletionQueryForMutateDocumentKey(msg tea.KeyMsg) (string, bool) {
	if m.buf == nil {
		return "", false
	}

	tmp := buffer.New(m.buf.Text(), buffer.Options{})
	tmp.SetCursor(m.buf.Cursor())
	if sel, ok := m.buf.Selection(); ok {
		tmp.SetSelection(sel)
	}

	switch {
	case key.Matches(msg, m.cfg.KeyMap.Backspace):
		tmp.DeleteBackward()
	case msg.Type == tea.KeySpace && !msg.Alt:
		tmp.InsertGrapheme(" ")
	case msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && !msg.Alt && !msg.Paste:
		tmp.InsertText(string(msg.Runes))
	default:
		return "", false
	}

	anchor := m.completionState.Anchor
	cursor := tmp.Cursor()
	if cursor.Row != anchor.Row || cursor.GraphemeCol < anchor.GraphemeCol {
		return "", true
	}
	return textInRange(tmp.Text(), buffer.Range{
		Start: anchor,
		End:   cursor,
	}), true
}

func (m *Model) recomputeCompletionQueryFromAnchor() {
	if m.buf == nil {
		return
	}

	state := m.completionState
	if !state.Visible {
		return
	}

	cursor := m.buf.Cursor()
	query := ""
	if cursor.Row == state.Anchor.Row && cursor.GraphemeCol >= state.Anchor.GraphemeCol {
		query = textInRange(m.buf.Text(), buffer.Range{
			Start: state.Anchor,
			End:   cursor,
		})
	}
	state.Query = query
	m.recomputeCompletionFilter(&state)
	m.completionState = state
}

func (m *Model) setCompletionQuery(query string) {
	state := m.completionState
	state.Query = query
	m.recomputeCompletionFilter(&state)
	m.completionState = state
}

func (m *Model) normalizeCompletionRuntimeState(state *CompletionState) {
	if state == nil {
		return
	}
	state.VisibleIndices = sanitizeCompletionVisibleIndices(state.VisibleIndices, len(state.Items))
	if len(state.VisibleIndices) == 0 {
		state.Selected = 0
		return
	}
	state.Selected = clampCompletionSelected(state.Selected, len(state.VisibleIndices))
}
