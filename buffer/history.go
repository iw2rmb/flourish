package buffer

type bufferSnapshot struct {
	text   string
	cursor Pos
	sel    selectionState
}

type historyState struct {
	undo []bufferSnapshot
	redo []bufferSnapshot
}

func (b *Buffer) snapshot() bufferSnapshot {
	return bufferSnapshot{
		text:   b.Text(),
		cursor: b.cursor,
		sel:    b.sel,
	}
}

func (b *Buffer) restore(s bufferSnapshot) {
	b.lines = splitLines(s.text)
	b.cursor = ClampPos(s.cursor, len(b.lines), b.lineLen)

	if !s.sel.active {
		b.sel = selectionState{}
		return
	}

	anchor := ClampPos(s.sel.anchor, len(b.lines), b.lineLen)
	end := ClampPos(s.sel.end, len(b.lines), b.lineLen)
	if NormalizeRange(Range{Start: anchor, End: end}).IsEmpty() {
		b.sel = selectionState{}
		return
	}
	b.sel = selectionState{active: true, anchor: anchor, end: end}
}

func (b *Buffer) recordUndo(prev bufferSnapshot) {
	limit := b.opt.HistoryLimit
	if limit <= 0 {
		return
	}

	b.hist.undo = append(b.hist.undo, prev)
	if len(b.hist.undo) > limit {
		b.hist.undo = b.hist.undo[len(b.hist.undo)-limit:]
	}
	b.hist.redo = nil
}

func (b *Buffer) CanUndo() bool { return len(b.hist.undo) > 0 }

func (b *Buffer) CanRedo() bool { return len(b.hist.redo) > 0 }

func (b *Buffer) Undo() bool {
	if len(b.hist.undo) == 0 {
		return false
	}

	cur := b.snapshot()
	change := b.beginChange(ChangeSourceLocal)

	i := len(b.hist.undo) - 1
	prev := b.hist.undo[i]
	b.hist.undo = b.hist.undo[:i]
	b.hist.redo = append(b.hist.redo, cur)

	b.restore(prev)
	b.version++
	if applied, ok := replacementAppliedEdit(cur.text, prev.text); ok {
		change.addAppliedEdit(applied)
	}
	b.commitChange(change)
	return true
}

func (b *Buffer) Redo() bool {
	if len(b.hist.redo) == 0 {
		return false
	}

	cur := b.snapshot()
	change := b.beginChange(ChangeSourceLocal)

	i := len(b.hist.redo) - 1
	next := b.hist.redo[i]
	b.hist.redo = b.hist.redo[:i]

	limit := b.opt.HistoryLimit
	if limit > 0 {
		b.hist.undo = append(b.hist.undo, cur)
		if len(b.hist.undo) > limit {
			b.hist.undo = b.hist.undo[len(b.hist.undo)-limit:]
		}
	}

	b.restore(next)
	b.version++
	if applied, ok := replacementAppliedEdit(cur.text, next.text); ok {
		change.addAppliedEdit(applied)
	}
	b.commitChange(change)
	return true
}
