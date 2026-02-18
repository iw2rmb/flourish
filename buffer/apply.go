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

	prev := b.snapshot()
	change := b.beginChange(ChangeSourceLocal)

	anyChanged := false
	lastCursor := b.cursor

	for _, e := range edits {
		nextCursor, applied, changed := b.replaceRange(e.Range, e.Text)
		if !changed {
			continue
		}
		anyChanged = true
		lastCursor = nextCursor
		change.addAppliedEdit(applied)
	}

	if !anyChanged {
		return
	}

	b.cursor = b.clampPos(lastCursor)
	b.sel = selectionState{}
	b.version++
	b.recordUndo(prev)
	b.commitChange(change)
}

// ApplyRemote applies remote edits in order and returns a change payload with
// baseline cursor/selection remap details.
//
// This phase introduces the public API contract. Advanced causality and
// remapping policy options are intentionally handled in later phases.
func (b *Buffer) ApplyRemote(edits []RemoteEdit, opts ApplyRemoteOptions) (ApplyRemoteResult, bool) {
	if len(edits) == 0 {
		return ApplyRemoteResult{}, false
	}

	// Option wiring is finalized in later phases; keep the shape stable now.
	_ = opts

	prev := b.snapshot()
	cursorBefore := b.cursor
	selectionBefore, selectionActiveBefore := b.Selection()
	change := b.beginChange(ChangeSourceRemote)

	anyChanged := false
	for _, e := range edits {
		_, applied, changed := b.replaceRange(e.Range, e.Text)
		if !changed {
			continue
		}
		anyChanged = true
		change.addAppliedEdit(applied)
	}
	if !anyChanged {
		return ApplyRemoteResult{}, false
	}

	remap := RemapReport{
		Cursor: b.remapClampedPoint(cursorBefore),
	}

	nextSelection := selectionState{}
	if selectionActiveBefore {
		remap.SelStart = b.remapClampedPoint(selectionBefore.Start)
		remap.SelEnd = b.remapClampedPoint(selectionBefore.End)
		if remap.SelStart.After == remap.SelEnd.After {
			remap.SelStart.Status = RemapInvalidated
			remap.SelEnd.Status = RemapInvalidated
		} else {
			nextSelection = selectionState{
				active: true,
				anchor: remap.SelStart.After,
				end:    remap.SelEnd.After,
			}
		}
	}

	b.cursor = remap.Cursor.After
	b.sel = nextSelection
	b.version++
	b.recordUndo(prev)
	b.commitChange(change)

	ch, _ := b.LastChange()
	return ApplyRemoteResult{
		Change: ch,
		Remap:  remap,
	}, true
}

func (b *Buffer) remapClampedPoint(before Pos) RemapPoint {
	after := b.clampPos(before)
	status := RemapUnchanged
	if after != before {
		status = RemapClamped
	}
	return RemapPoint{
		Before: before,
		After:  after,
		Status: status,
	}
}
