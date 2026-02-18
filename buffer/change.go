package buffer

// ChangeSource identifies where a change originated.
type ChangeSource uint8

const (
	ChangeSourceLocal ChangeSource = iota
	ChangeSourceRemote
)

// SelectionState captures normalized selection state at a point in time.
type SelectionState struct {
	Active bool
	Range  Range
}

// AppliedEdit describes one effective edit in a change transaction.
type AppliedEdit struct {
	RangeBefore Range
	RangeAfter  Range
	InsertText  string
	DeletedText string
}

// Change is a normalized, versioned mutation payload.
type Change struct {
	Source          ChangeSource
	VersionBefore   uint64
	VersionAfter    uint64
	CursorBefore    Pos
	CursorAfter     Pos
	SelectionBefore SelectionState
	SelectionAfter  SelectionState
	AppliedEdits    []AppliedEdit
}

type changeBuilder struct {
	source          ChangeSource
	versionBefore   uint64
	cursorBefore    Pos
	selectionBefore SelectionState
	appliedEdits    []AppliedEdit
}

// LastChange returns the most recent effective change.
func (b *Buffer) LastChange() (Change, bool) {
	if !b.hasLastChange {
		return Change{}, false
	}
	return cloneChange(b.lastChange), true
}

func cloneChange(in Change) Change {
	out := in
	out.AppliedEdits = append([]AppliedEdit(nil), in.AppliedEdits...)
	return out
}

func selectionStateFromInternal(sel selectionState) SelectionState {
	if !sel.active {
		return SelectionState{}
	}
	r := NormalizeRange(Range{Start: sel.anchor, End: sel.end})
	if r.IsEmpty() {
		return SelectionState{}
	}
	return SelectionState{
		Active: true,
		Range:  r,
	}
}

func (b *Buffer) beginChange(source ChangeSource) changeBuilder {
	return changeBuilder{
		source:          source,
		versionBefore:   b.version,
		cursorBefore:    b.cursor,
		selectionBefore: selectionStateFromInternal(b.sel),
	}
}

func (cb *changeBuilder) addAppliedEdit(edit AppliedEdit) {
	edit.RangeBefore = NormalizeRange(edit.RangeBefore)
	edit.RangeAfter = NormalizeRange(edit.RangeAfter)
	cb.appliedEdits = append(cb.appliedEdits, edit)
}

func (b *Buffer) commitChange(cb changeBuilder) {
	if b.version == cb.versionBefore {
		return
	}
	b.lastChange = Change{
		Source:          cb.source,
		VersionBefore:   cb.versionBefore,
		VersionAfter:    b.version,
		CursorBefore:    cb.cursorBefore,
		CursorAfter:     b.cursor,
		SelectionBefore: cb.selectionBefore,
		SelectionAfter:  selectionStateFromInternal(b.sel),
		AppliedEdits:    append([]AppliedEdit(nil), cb.appliedEdits...),
	}
	b.hasLastChange = true
}

func replacementAppliedEdit(beforeText, afterText string) (AppliedEdit, bool) {
	if beforeText == afterText {
		return AppliedEdit{}, false
	}
	return AppliedEdit{
		RangeBefore: fullDocumentRange(beforeText),
		RangeAfter:  fullDocumentRange(afterText),
		InsertText:  afterText,
		DeletedText: beforeText,
	}, true
}

func fullDocumentRange(text string) Range {
	lines := splitLines(text)
	lastRow := len(lines) - 1
	lastCol := 0
	if lastRow >= 0 {
		lastCol = len(lines[lastRow])
	}
	return Range{
		Start: Pos{Row: 0, GraphemeCol: 0},
		End:   Pos{Row: lastRow, GraphemeCol: lastCol},
	}
}
