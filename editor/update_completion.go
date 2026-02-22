package editor

import (
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
	"github.com/iw2rmb/flourish/internal/grapheme"
)

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

	anchor := m.completionState.Anchor
	cursor := m.buf.Cursor()
	sel, hasSel := m.buf.Selection()

	// predictInsert predicts the query after replacing the selection (or
	// inserting at cursor when there is no selection) with text.
	predictInsert := func(text string) (string, bool) {
		insertAt := cursor
		if hasSel {
			insertAt = sel.Start
		}
		if insertAt.Row != anchor.Row || insertAt.GraphemeCol < anchor.GraphemeCol {
			return "", true
		}
		prefix := m.buf.TextInRange(buffer.Range{Start: anchor, End: insertAt})
		return prefix + text, true
	}

	switch {
	case key.Matches(msg, m.cfg.KeyMap.Backspace):
		if hasSel {
			// Selection delete: cursor lands at selection start.
			insertAt := sel.Start
			if insertAt.Row != anchor.Row || insertAt.GraphemeCol < anchor.GraphemeCol {
				return "", true
			}
			return m.buf.TextInRange(buffer.Range{Start: anchor, End: insertAt}), true
		}
		// Simple backspace: one grapheme removed before cursor.
		if cursor.Row != anchor.Row {
			return "", true
		}
		if cursor.GraphemeCol <= 0 {
			// At line start: backspace merges with previous line.
			return "", true
		}
		newCol := cursor.GraphemeCol - 1
		if newCol < anchor.GraphemeCol {
			return "", true
		}
		return m.buf.TextInRange(buffer.Range{
			Start: anchor,
			End:   buffer.Pos{Row: cursor.Row, GraphemeCol: newCol},
		}), true

	case msg.Type == tea.KeySpace && !msg.Alt:
		return predictInsert(" ")

	case msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && !msg.Alt && !msg.Paste:
		return predictInsert(string(msg.Runes))

	default:
		return "", false
	}
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
		query = m.buf.TextInRange(buffer.Range{
			Start: state.Anchor,
			End:   cursor,
		})
	}
	state.Query = query
	m.recomputeCompletionFilter(&state)
	m.completionState = state
	m.completionFilterClean = true
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

func completionAnchorTokenBounds(rowClusters []string, anchorCol int) (startCol int, endCol int) {
	anchorCol = clampInt(anchorCol, 0, len(rowClusters))
	startCol = anchorCol
	for startCol > 0 && isCompletionTokenGrapheme(rowClusters[startCol-1]) {
		startCol--
	}

	endCol = anchorCol
	for endCol < len(rowClusters) && isCompletionTokenGrapheme(rowClusters[endCol]) {
		endCol++
	}
	return startCol, endCol
}

func isCompletionTokenGrapheme(g string) bool {
	if utf8.RuneCountInString(g) != 1 {
		return false
	}
	r, _ := utf8.DecodeRuneInString(g)
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
