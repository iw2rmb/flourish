package buffer

import (
	"reflect"
	"strings"
	"testing"
)

func FuzzBuffer_ApplyRemoteRandomSequences(f *testing.F) {
	seeds := [][]byte{
		{},
		{0},
		{1, 2, 3, 4, 5},
		{255, 0, 128, 64, 32, 16, 8, 4, 2, 1},
		[]byte("overlap-seed"),
		[]byte("multiline\nseed"),
		[]byte("unicode-seed-👨‍👩‍👧‍👦"),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		tc := decodeApplyRemoteFuzzCase(data)

		b1 := buildApplyRemoteFuzzBuffer(tc)
		before1 := snapshotApplyRemoteFuzzState(b1)
		opts1 := buildApplyRemoteFuzzOptions(tc, before1.Version)
		res1, changed1 := b1.ApplyRemote(tc.edits, opts1)
		after1 := snapshotApplyRemoteFuzzState(b1)

		b2 := buildApplyRemoteFuzzBuffer(tc)
		before2 := snapshotApplyRemoteFuzzState(b2)
		opts2 := buildApplyRemoteFuzzOptions(tc, before2.Version)
		res2, changed2 := b2.ApplyRemote(tc.edits, opts2)
		after2 := snapshotApplyRemoteFuzzState(b2)

		if !reflect.DeepEqual(before1, before2) {
			t.Fatalf("pre-state mismatch between identical runs: %#v vs %#v", before1, before2)
		}
		if opts1 != opts2 {
			t.Fatalf("options mismatch between identical runs: %#v vs %#v", opts1, opts2)
		}
		if changed1 != changed2 {
			t.Fatalf("changed mismatch between identical runs: %v vs %v", changed1, changed2)
		}
		if !reflect.DeepEqual(res1, res2) {
			t.Fatalf("result mismatch between identical runs: %#v vs %#v", res1, res2)
		}
		if !reflect.DeepEqual(after1, after2) {
			t.Fatalf("post-state mismatch between identical runs: %#v vs %#v", after1, after2)
		}

		assertApplyRemoteFuzzInvariants(t, b1, before1, after1, res1, changed1, opts1)
	})
}

type applyRemoteFuzzCase struct {
	initialText         string
	cursor              Pos
	selection           Range
	selectionActive     bool
	edits               []RemoteEdit
	clampMode           OffsetClampMode
	versionMismatchMode VersionMismatchMode
	forceMismatch       bool
	mismatchDelta       uint64
}

type applyRemoteFuzzSnapshot struct {
	Text            string
	Cursor          Pos
	Selection       Range
	SelectionActive bool
	Version         uint64
	LastChange      Change
	HasLastChange   bool
}

type fuzzByteReader struct {
	data []byte
	idx  int
}

func decodeApplyRemoteFuzzCase(data []byte) applyRemoteFuzzCase {
	r := fuzzByteReader{data: data}

	initialText := fuzzDocText(&r, 1+r.nextInt(4), 6)
	seedBuffer := New(initialText, Options{})

	cursor := fuzzPosFromBuffer(seedBuffer, &r)
	selectionActive := r.nextBool()
	selection := Range{
		Start: fuzzPosFromBuffer(seedBuffer, &r),
		End:   fuzzPosFromBuffer(seedBuffer, &r),
	}

	editCount := r.nextInt(9)
	edits := make([]RemoteEdit, 0, editCount)
	for i := 0; i < editCount; i++ {
		edits = append(edits, RemoteEdit{
			Range: Range{
				Start: fuzzPosFromBuffer(seedBuffer, &r),
				End:   fuzzPosFromBuffer(seedBuffer, &r),
			},
			Text: fuzzEditText(&r),
			OpID: fuzzOpID(&r),
		})
	}

	clampMode := OffsetClamp
	if r.nextBool() {
		clampMode = OffsetError
	}
	versionMismatchMode := VersionMismatchReject
	if r.nextBool() {
		versionMismatchMode = VersionMismatchForceApply
	}

	return applyRemoteFuzzCase{
		initialText:         initialText,
		cursor:              cursor,
		selection:           selection,
		selectionActive:     selectionActive,
		edits:               edits,
		clampMode:           clampMode,
		versionMismatchMode: versionMismatchMode,
		forceMismatch:       r.nextBool(),
		mismatchDelta:       uint64(1 + r.nextInt(4)),
	}
}

func buildApplyRemoteFuzzBuffer(tc applyRemoteFuzzCase) *Buffer {
	b := New(tc.initialText, Options{})
	b.SetCursor(tc.cursor)
	if tc.selectionActive {
		b.SetSelection(tc.selection)
	}
	return b
}

func buildApplyRemoteFuzzOptions(tc applyRemoteFuzzCase, baseVersion uint64) ApplyRemoteOptions {
	base := baseVersion
	if tc.forceMismatch {
		base += tc.mismatchDelta
	}

	return ApplyRemoteOptions{
		BaseVersion: base,
		ClampPolicy: ConvertPolicy{
			ClampMode:   tc.clampMode,
			NewlineMode: NewlineAsSingleRune,
		},
		VersionMismatchMode: tc.versionMismatchMode,
	}
}

func snapshotApplyRemoteFuzzState(b *Buffer) applyRemoteFuzzSnapshot {
	selection, selectionActive := b.Selection()
	lastChange, hasLastChange := b.LastChange()
	return applyRemoteFuzzSnapshot{
		Text:            b.Text(),
		Cursor:          b.Cursor(),
		Selection:       selection,
		SelectionActive: selectionActive,
		Version:         b.Version(),
		LastChange:      lastChange,
		HasLastChange:   hasLastChange,
	}
}

func assertApplyRemoteFuzzInvariants(t *testing.T, b *Buffer, before, after applyRemoteFuzzSnapshot, res ApplyRemoteResult, changed bool, opts ApplyRemoteOptions) {
	t.Helper()

	if opts.VersionMismatchMode == VersionMismatchReject && opts.BaseVersion != before.Version {
		if changed {
			t.Fatalf("reject mode with mismatched version must not change state")
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("reject mode with mismatched version changed state: before=%#v after=%#v", before, after)
		}
		return
	}

	if !changed {
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("changed=false but state mutated: before=%#v after=%#v", before, after)
		}
		var zero ApplyRemoteResult
		if !reflect.DeepEqual(res, zero) {
			t.Fatalf("changed=false must return zero result, got %#v", res)
		}
		return
	}

	if after.Version != before.Version+1 {
		t.Fatalf("version increment mismatch: before=%d after=%d", before.Version, after.Version)
	}
	if got, want := res.Change.Source, ChangeSourceRemote; got != want {
		t.Fatalf("change source=%v, want %v", got, want)
	}
	if got, want := res.Change.VersionBefore, before.Version; got != want {
		t.Fatalf("result version before=%d, want %d", got, want)
	}
	if got, want := res.Change.VersionAfter, after.Version; got != want {
		t.Fatalf("result version after=%d, want %d", got, want)
	}
	if got, want := after.Cursor, res.Remap.Cursor.After; got != want {
		t.Fatalf("cursor after=%v, remap after=%v", got, want)
	}
	if len(res.Change.AppliedEdits) == 0 {
		t.Fatalf("changed=true requires at least one applied edit")
	}
	if !isValidRemapStatus(res.Remap.Cursor.Status) || !isValidRemapStatus(res.Remap.SelStart.Status) || !isValidRemapStatus(res.Remap.SelEnd.Status) {
		t.Fatalf("invalid remap status in report: %+v", res.Remap)
	}

	last, ok := b.LastChange()
	if !ok {
		t.Fatalf("changed=true expected last change")
	}
	if !reflect.DeepEqual(last, res.Change) {
		t.Fatalf("last change mismatch with result: last=%#v result=%#v", last, res.Change)
	}

	if before.SelectionActive && res.Remap.SelStart.Status == RemapInvalidated && res.Remap.SelEnd.Status == RemapInvalidated && after.SelectionActive {
		t.Fatalf("invalidated selection endpoints must clear selection")
	}
}

func isValidRemapStatus(status RemapStatus) bool {
	switch status {
	case RemapUnchanged, RemapMoved, RemapClamped, RemapInvalidated:
		return true
	default:
		return false
	}
}

func (r *fuzzByteReader) nextByte() byte {
	if len(r.data) == 0 {
		return 0
	}
	b := r.data[r.idx%len(r.data)]
	r.idx++
	return b
}

func (r *fuzzByteReader) nextBool() bool {
	return r.nextByte()&1 == 1
}

func (r *fuzzByteReader) nextInt(max int) int {
	if max <= 0 {
		return 0
	}
	return int(r.nextByte()) % max
}

func fuzzPosFromBuffer(b *Buffer, r *fuzzByteReader) Pos {
	policy := ConvertPolicy{
		ClampMode:   OffsetClamp,
		NewlineMode: NewlineAsSingleRune,
	}
	maxRune := b.docLen(offsetUnitRune)
	off := r.nextInt(maxRune + 2)
	pos, ok := b.PosFromRuneOffset(off, policy)
	if !ok {
		pos = Pos{}
	}

	if r.nextBool() {
		pos.Row += r.nextInt(4)
	}
	if r.nextBool() {
		pos.GraphemeCol += r.nextInt(8)
	}
	return pos
}

func fuzzOpID(r *fuzzByteReader) string {
	if r.nextBool() {
		return ""
	}
	return "op"
}

func fuzzEditText(r *fuzzByteReader) string {
	if r.nextInt(4) == 0 {
		return ""
	}
	return fuzzDocText(r, 1+r.nextInt(3), 4)
}

func fuzzDocText(r *fuzzByteReader, lineCount, maxClustersPerLine int) string {
	if lineCount <= 0 {
		lineCount = 1
	}
	if maxClustersPerLine < 0 {
		maxClustersPerLine = 0
	}

	clusters := []string{"a", "b", "c", "x", " ", "é", "e\u0301", "中", "👨‍👩‍👧‍👦"}
	lines := make([]string, 0, lineCount)
	for i := 0; i < lineCount; i++ {
		n := r.nextInt(maxClustersPerLine + 1)
		var sb strings.Builder
		for j := 0; j < n; j++ {
			sb.WriteString(clusters[r.nextInt(len(clusters))])
		}
		lines = append(lines, sb.String())
	}
	return strings.Join(lines, "\n")
}
