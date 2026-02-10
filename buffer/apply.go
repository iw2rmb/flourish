package buffer

// Apply applies a sequence of text edits in order. Each edit's range is
// interpreted against the buffer state at the time that edit is applied.
//
// v0 semantics:
// - Edit ranges are clamped into current document bounds.
// - Empty range + non-empty text inserts.
// - Cursor moves to the end of the last applied (effective) edit.
// - Selection is cleared if any edit applies.
func (b *Buffer) Apply(edits ...TextEdit) {
	if len(edits) == 0 {
		return
	}

	prevCursor := b.cursor
	prevSel := b.sel

	anyChanged := false
	lastCursor := b.cursor

	for _, e := range edits {
		nextCursor, changed := b.replaceRange(e.Range, e.Text)
		if !changed {
			continue
		}
		anyChanged = true
		lastCursor = nextCursor
	}

	if !anyChanged {
		return
	}

	b.cursor = b.clampPos(lastCursor)
	b.sel = selectionState{}

	if prevCursor == b.cursor && selectionStateEqual(prevSel, b.sel) {
		// Defensive: replaceRange changed text, so this should be unreachable, but
		// keep version semantics consistent.
		b.version++
		return
	}
	b.version++
}
