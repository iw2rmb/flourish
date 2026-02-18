package buffer

import "unicode/utf8"

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
// deterministic cursor/selection remap details.
func (b *Buffer) ApplyRemote(edits []RemoteEdit, opts ApplyRemoteOptions) (ApplyRemoteResult, bool) {
	if len(edits) == 0 {
		return ApplyRemoteResult{}, false
	}
	if !validVersionMismatchMode(opts.VersionMismatchMode) {
		return ApplyRemoteResult{}, false
	}
	if !validNewlineMode(opts.ClampPolicy.NewlineMode) {
		return ApplyRemoteResult{}, false
	}
	if !validRemoteClampMode(opts.ClampPolicy.ClampMode) {
		return ApplyRemoteResult{}, false
	}
	if opts.BaseVersion != b.Version() && opts.VersionMismatchMode == VersionMismatchReject {
		return ApplyRemoteResult{}, false
	}

	prev := b.snapshot()
	cursorBefore := b.cursor
	selectionBefore, selectionActiveBefore := b.Selection()
	change := b.beginChange(ChangeSourceRemote)

	cursorRemap, ok := b.newRemoteRemapTracker(cursorBefore)
	if !ok {
		return ApplyRemoteResult{}, false
	}

	var selStartRemap remoteRemapTracker
	var selEndRemap remoteRemapTracker
	if selectionActiveBefore {
		selStartRemap, ok = b.newRemoteRemapTracker(selectionBefore.Start)
		if !ok {
			return ApplyRemoteResult{}, false
		}
		selEndRemap, ok = b.newRemoteRemapTracker(selectionBefore.End)
		if !ok {
			return ApplyRemoteResult{}, false
		}
	}

	anyChanged := false
	for _, e := range edits {
		r, ok := b.normalizeRemoteRangeForMode(e.Range, opts.ClampPolicy.ClampMode)
		if !ok {
			b.restore(prev)
			return ApplyRemoteResult{}, false
		}

		startOff, ok := b.RuneOffsetFromPos(r.Start, remoteOffsetErrorPolicy())
		if !ok {
			b.restore(prev)
			return ApplyRemoteResult{}, false
		}
		endOff, ok := b.RuneOffsetFromPos(r.End, remoteOffsetErrorPolicy())
		if !ok {
			b.restore(prev)
			return ApplyRemoteResult{}, false
		}
		insertLen := utf8.RuneCountInString(e.Text)

		_, applied, changed := b.replaceRange(r, e.Text)
		if !changed {
			continue
		}
		anyChanged = true
		change.addAppliedEdit(applied)
		cursorRemap.applyEdit(startOff, endOff, insertLen)
		if selectionActiveBefore {
			selStartRemap.applyEdit(startOff, endOff, insertLen)
			selEndRemap.applyEdit(startOff, endOff, insertLen)
		}
	}
	if !anyChanged {
		return ApplyRemoteResult{}, false
	}

	cursorPoint, ok := b.finalizeRemoteRemapPoint(cursorRemap)
	if !ok {
		b.restore(prev)
		return ApplyRemoteResult{}, false
	}
	remap := RemapReport{Cursor: cursorPoint}

	nextSelection := selectionState{}
	if selectionActiveBefore {
		remap.SelStart, ok = b.finalizeRemoteRemapPoint(selStartRemap)
		if !ok {
			b.restore(prev)
			return ApplyRemoteResult{}, false
		}
		remap.SelEnd, ok = b.finalizeRemoteRemapPoint(selEndRemap)
		if !ok {
			b.restore(prev)
			return ApplyRemoteResult{}, false
		}
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

type remoteRemapTracker struct {
	before Pos
	off    int
	status RemapStatus
}

func validVersionMismatchMode(mode VersionMismatchMode) bool {
	return mode == VersionMismatchReject || mode == VersionMismatchForceApply
}

func validRemoteClampMode(mode OffsetClampMode) bool {
	return mode == OffsetError || mode == OffsetClamp
}

func remoteOffsetErrorPolicy() ConvertPolicy {
	return ConvertPolicy{
		ClampMode:   OffsetError,
		NewlineMode: NewlineAsSingleRune,
	}
}

func remoteOffsetClampPolicy() ConvertPolicy {
	return ConvertPolicy{
		ClampMode:   OffsetClamp,
		NewlineMode: NewlineAsSingleRune,
	}
}

func (b *Buffer) normalizeRemoteRangeForMode(r Range, mode OffsetClampMode) (Range, bool) {
	switch mode {
	case OffsetError:
		start, ok := b.normalizePosForMode(r.Start, OffsetError)
		if !ok {
			return Range{}, false
		}
		end, ok := b.normalizePosForMode(r.End, OffsetError)
		if !ok {
			return Range{}, false
		}
		return NormalizeRange(Range{Start: start, End: end}), true
	case OffsetClamp:
		return NormalizeRange(Range{
			Start: b.clampPos(r.Start),
			End:   b.clampPos(r.End),
		}), true
	default:
		return Range{}, false
	}
}

func (b *Buffer) newRemoteRemapTracker(before Pos) (remoteRemapTracker, bool) {
	off, ok := b.RuneOffsetFromPos(before, remoteOffsetErrorPolicy())
	if !ok {
		return remoteRemapTracker{}, false
	}
	return remoteRemapTracker{
		before: before,
		off:    off,
		status: RemapUnchanged,
	}, true
}

func (t *remoteRemapTracker) applyEdit(start, end, insertLen int) {
	nextOff, effect := transformRemoteOffset(t.off, start, end, insertLen)
	t.off = nextOff
	switch {
	case t.status == RemapClamped:
		return
	case effect == RemapClamped:
		t.status = RemapClamped
	case effect == RemapMoved && t.status == RemapUnchanged:
		t.status = RemapMoved
	}
}

func transformRemoteOffset(off, start, end, insertLen int) (int, RemapStatus) {
	if start == end {
		if off < start || insertLen == 0 {
			return off, RemapUnchanged
		}
		return off + insertLen, RemapMoved
	}

	if off < start {
		return off, RemapUnchanged
	}

	deletedLen := end - start
	delta := insertLen - deletedLen
	if off > end {
		if delta == 0 {
			return off, RemapUnchanged
		}
		return off + delta, RemapMoved
	}

	if off == start {
		return off, RemapUnchanged
	}
	if off == end {
		if delta == 0 {
			return off, RemapUnchanged
		}
		return off + delta, RemapMoved
	}

	return start + insertLen, RemapClamped
}

func (b *Buffer) finalizeRemoteRemapPoint(t remoteRemapTracker) (RemapPoint, bool) {
	after, ok := b.PosFromRuneOffset(t.off, remoteOffsetErrorPolicy())
	status := t.status
	if !ok {
		after, ok = b.PosFromRuneOffset(t.off, remoteOffsetClampPolicy())
		if !ok {
			return RemapPoint{}, false
		}
		status = RemapClamped
	}
	if status == RemapUnchanged && after != t.before {
		status = RemapMoved
	}
	return RemapPoint{
		Before: t.before,
		After:  after,
		Status: status,
	}, true
}
