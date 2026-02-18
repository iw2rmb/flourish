package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
	"github.com/iw2rmb/flourish/internal/grapheme"
)

type localMutationOp func(*Model)

func (m Model) updateKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if !m.focused || m.buf == nil {
		return m, nil
	}

	batch, mutations := (&m).buildIntentsFromKey(msg)
	mode := normalizeMutationMode(m.cfg.MutationMode)

	applyLocally := true
	switch mode {
	case MutateInEditor:
		applyLocally = true
	case EmitIntentsOnly:
		if len(batch.Intents) > 0 && m.cfg.OnIntent != nil {
			m.cfg.OnIntent(batch)
		}
		applyLocally = false
	case EmitIntentsAndMutate:
		applyLocally = true
		if len(batch.Intents) > 0 && m.cfg.OnIntent != nil {
			decision := m.cfg.OnIntent(batch)
			applyLocally = decision.ApplyLocally
		}
	}

	if applyLocally {
		for _, op := range mutations {
			op(&m)
		}
	}

	return m, nil
}

func (m *Model) buildIntentsFromKey(msg tea.KeyMsg) (IntentBatch, []localMutationOp) {
	batch := IntentBatch{}
	mutations := make([]localMutationOp, 0, 1)
	before := editorStateFromBuffer(m.buf)
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
