package buffer

import "testing"

func remoteOpts(base uint64) ApplyRemoteOptions {
	return ApplyRemoteOptions{
		BaseVersion: base,
		ClampPolicy: ConvertPolicy{
			ClampMode:   OffsetClamp,
			NewlineMode: NewlineAsSingleRune,
		},
		VersionMismatchMode: VersionMismatchReject,
	}
}

func TestBuffer_ApplyRemote_APIAndRemoteChange(t *testing.T) {
	b := New("hello", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 2})
	v := b.Version()

	res, changed := b.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 0},
			},
			Text: "X",
			OpID: "op-1",
		},
	}, remoteOpts(b.Version()))
	if !changed {
		t.Fatalf("expected changed=true")
	}

	if got, want := b.Text(), "Xhello"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Version(), v+1; got != want {
		t.Fatalf("version=%d, want %d", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 3}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}

	if got, want := res.Change.Source, ChangeSourceRemote; got != want {
		t.Fatalf("change source=%v, want %v", got, want)
	}
	if got, want := len(res.Change.AppliedEdits), 1; got != want {
		t.Fatalf("applied edits=%d, want %d", got, want)
	}
	if got, want := res.Remap.Cursor.Before, (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor before=%v, want %v", got, want)
	}
	if got, want := res.Remap.Cursor.After, (Pos{Row: 0, GraphemeCol: 3}); got != want {
		t.Fatalf("cursor after=%v, want %v", got, want)
	}
	if got, want := res.Remap.Cursor.Status, RemapMoved; got != want {
		t.Fatalf("cursor status=%v, want %v", got, want)
	}
}

func TestBuffer_ApplyRemote_NoOpDoesNotBumpVersion(t *testing.T) {
	b := New("a", Options{})
	v := b.Version()

	res, changed := b.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 0},
			},
			Text: "",
		},
	}, remoteOpts(b.Version()))
	if changed {
		t.Fatalf("expected changed=false")
	}
	if got := b.Version(); got != v {
		t.Fatalf("version=%d, want %d", got, v)
	}
	if got := len(res.Change.AppliedEdits); got != 0 {
		t.Fatalf("expected no applied edits in zero result, got %d", got)
	}
	if res.Change.VersionAfter != 0 || res.Change.VersionBefore != 0 {
		t.Fatalf("expected zero-value change versions in no-op result")
	}
	if res.Remap != (RemapReport{}) {
		t.Fatalf("expected zero-value remap report in no-op result")
	}
}

func TestBuffer_ApplyRemote_CursorClampStatus(t *testing.T) {
	b := New("abc", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 2})

	res, changed := b.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 3},
			},
			Text: "",
		},
	}, remoteOpts(b.Version()))
	if !changed {
		t.Fatalf("expected changed=true")
	}

	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 0}); got != want {
		t.Fatalf("cursor=%v, want %v", got, want)
	}
	if got, want := res.Remap.Cursor.Status, RemapClamped; got != want {
		t.Fatalf("cursor status=%v, want %v", got, want)
	}
}

func TestBuffer_ApplyRemote_SelectionInvalidatedWhenCollapsed(t *testing.T) {
	b := New("abcd", Options{})
	b.SetSelection(Range{
		Start: Pos{Row: 0, GraphemeCol: 1},
		End:   Pos{Row: 0, GraphemeCol: 3},
	})

	res, changed := b.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 4},
			},
			Text: "",
		},
	}, remoteOpts(b.Version()))
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if _, ok := b.Selection(); ok {
		t.Fatalf("expected selection cleared")
	}
	if got, want := res.Remap.SelStart.Status, RemapInvalidated; got != want {
		t.Fatalf("sel start status=%v, want %v", got, want)
	}
	if got, want := res.Remap.SelEnd.Status, RemapInvalidated; got != want {
		t.Fatalf("sel end status=%v, want %v", got, want)
	}
}

func TestBuffer_ApplyRemote_OrderedOverlapDeterministic(t *testing.T) {
	ordered := []RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 1},
				End:   Pos{Row: 0, GraphemeCol: 4},
			},
			Text: "X",
		},
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 1},
				End:   Pos{Row: 0, GraphemeCol: 3},
			},
			Text: "YZ",
		},
	}

	b1 := New("abcdef", Options{})
	b1.SetCursor(Pos{Row: 0, GraphemeCol: 4})
	r1, changed := b1.ApplyRemote(ordered, remoteOpts(b1.Version()))
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if got, want := b1.Text(), "aYZf"; got != want {
		t.Fatalf("ordered text=%q, want %q", got, want)
	}
	if got, want := b1.Cursor(), (Pos{Row: 0, GraphemeCol: 3}); got != want {
		t.Fatalf("ordered cursor=%v, want %v", got, want)
	}
	if got, want := r1.Remap.Cursor.Status, RemapClamped; got != want {
		t.Fatalf("ordered status=%v, want %v", got, want)
	}

	b2 := New("abcdef", Options{})
	b2.SetCursor(Pos{Row: 0, GraphemeCol: 4})
	reversed := []RemoteEdit{ordered[1], ordered[0]}
	if _, changed := b2.ApplyRemote(reversed, remoteOpts(b2.Version())); !changed {
		t.Fatalf("expected changed=true for reversed order")
	}
	if got, want := b2.Text(), "aXef"; got != want {
		t.Fatalf("reversed text=%q, want %q", got, want)
	}

	b3 := New("abcdef", Options{})
	b3.SetCursor(Pos{Row: 0, GraphemeCol: 4})
	r3, changed := b3.ApplyRemote(ordered, remoteOpts(b3.Version()))
	if !changed {
		t.Fatalf("expected changed=true for repeated ordered run")
	}
	if got, want := b3.Text(), "aYZf"; got != want {
		t.Fatalf("repeated ordered text=%q, want %q", got, want)
	}
	if got, want := r3.Remap.Cursor, r1.Remap.Cursor; got != want {
		t.Fatalf("repeated ordered remap cursor=%v, want %v", got, want)
	}
}

func TestBuffer_ApplyRemote_CursorRemapStatusMatrix(t *testing.T) {
	tests := []struct {
		name       string
		cursor     Pos
		edit       RemoteEdit
		wantAfter  Pos
		wantStatus RemapStatus
	}{
		{
			name:   "unchanged-when-edit-after",
			cursor: Pos{Row: 0, GraphemeCol: 1},
			edit: RemoteEdit{
				Range: Range{
					Start: Pos{Row: 0, GraphemeCol: 4},
					End:   Pos{Row: 0, GraphemeCol: 6},
				},
				Text: "",
			},
			wantAfter:  Pos{Row: 0, GraphemeCol: 1},
			wantStatus: RemapUnchanged,
		},
		{
			name:   "moved-when-edit-before",
			cursor: Pos{Row: 0, GraphemeCol: 3},
			edit: RemoteEdit{
				Range: Range{
					Start: Pos{Row: 0, GraphemeCol: 1},
					End:   Pos{Row: 0, GraphemeCol: 1},
				},
				Text: "ZZ",
			},
			wantAfter:  Pos{Row: 0, GraphemeCol: 5},
			wantStatus: RemapMoved,
		},
		{
			name:   "clamped-when-edit-covers-point",
			cursor: Pos{Row: 0, GraphemeCol: 3},
			edit: RemoteEdit{
				Range: Range{
					Start: Pos{Row: 0, GraphemeCol: 2},
					End:   Pos{Row: 0, GraphemeCol: 5},
				},
				Text: "",
			},
			wantAfter:  Pos{Row: 0, GraphemeCol: 2},
			wantStatus: RemapClamped,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := New("abcdef", Options{})
			b.SetCursor(tc.cursor)
			res, changed := b.ApplyRemote([]RemoteEdit{tc.edit}, remoteOpts(b.Version()))
			if !changed {
				t.Fatalf("expected changed=true")
			}
			if got := b.Cursor(); got != tc.wantAfter {
				t.Fatalf("cursor=%v, want %v", got, tc.wantAfter)
			}
			if got := res.Remap.Cursor.Status; got != tc.wantStatus {
				t.Fatalf("status=%v, want %v", got, tc.wantStatus)
			}
		})
	}
}

func TestBuffer_ApplyRemote_SelectionEndpointsRemap(t *testing.T) {
	b := New("abcdef", Options{})
	b.SetSelection(Range{
		Start: Pos{Row: 0, GraphemeCol: 2},
		End:   Pos{Row: 0, GraphemeCol: 5},
	})

	res, changed := b.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 1},
				End:   Pos{Row: 0, GraphemeCol: 1},
			},
			Text: "ZZ",
		},
	}, remoteOpts(b.Version()))
	if !changed {
		t.Fatalf("expected changed=true")
	}

	selection, ok := b.Selection()
	if !ok {
		t.Fatalf("expected active selection")
	}
	if got, want := selection.Start, (Pos{Row: 0, GraphemeCol: 4}); got != want {
		t.Fatalf("selection start=%v, want %v", got, want)
	}
	if got, want := selection.End, (Pos{Row: 0, GraphemeCol: 7}); got != want {
		t.Fatalf("selection end=%v, want %v", got, want)
	}
	if got, want := res.Remap.SelStart.Status, RemapMoved; got != want {
		t.Fatalf("sel start status=%v, want %v", got, want)
	}
	if got, want := res.Remap.SelEnd.Status, RemapMoved; got != want {
		t.Fatalf("sel end status=%v, want %v", got, want)
	}

	b2 := New("abcdef", Options{})
	b2.SetSelection(Range{
		Start: Pos{Row: 0, GraphemeCol: 2},
		End:   Pos{Row: 0, GraphemeCol: 5},
	})
	res2, changed := b2.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 1},
				End:   Pos{Row: 0, GraphemeCol: 4},
			},
			Text: "Q",
		},
	}, remoteOpts(b2.Version()))
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if got, want := res2.Remap.SelStart.Status, RemapClamped; got != want {
		t.Fatalf("sel start status=%v, want %v", got, want)
	}
	if got, want := res2.Remap.SelEnd.Status, RemapMoved; got != want {
		t.Fatalf("sel end status=%v, want %v", got, want)
	}
}

func TestBuffer_ApplyRemote_BaseVersionMismatchPolicies(t *testing.T) {
	b := New("abc", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 1})
	version := b.Version()

	edit := []RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 0},
			},
			Text: "X",
		},
	}

	if _, changed := b.ApplyRemote(edit, remoteOpts(0)); changed {
		t.Fatalf("expected changed=false for mismatch reject")
	}
	if got, want := b.Text(), "abc"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got := b.Version(); got != version {
		t.Fatalf("version=%d, want %d", got, version)
	}

	force := remoteOpts(0)
	force.VersionMismatchMode = VersionMismatchForceApply
	if _, changed := b.ApplyRemote(edit, force); !changed {
		t.Fatalf("expected changed=true for mismatch force apply")
	}
	if got, want := b.Text(), "Xabc"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}

	b2 := New("abc", Options{})
	if _, changed := b2.ApplyRemote(edit, remoteOpts(b2.Version())); !changed {
		t.Fatalf("expected changed=true for matching base version")
	}
}

func TestBuffer_ApplyRemote_InvalidOptionsRejected(t *testing.T) {
	b := New("abc", Options{})
	edit := []RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 0},
			},
			Text: "X",
		},
	}

	invalidMismatch := remoteOpts(b.Version())
	invalidMismatch.VersionMismatchMode = VersionMismatchMode(99)
	if _, changed := b.ApplyRemote(edit, invalidMismatch); changed {
		t.Fatalf("expected changed=false for invalid mismatch mode")
	}

	invalidClamp := remoteOpts(b.Version())
	invalidClamp.ClampPolicy.ClampMode = OffsetClampMode(99)
	if _, changed := b.ApplyRemote(edit, invalidClamp); changed {
		t.Fatalf("expected changed=false for invalid clamp mode")
	}

	invalidNewline := remoteOpts(b.Version())
	invalidNewline.ClampPolicy.NewlineMode = NewlineMode(99)
	if _, changed := b.ApplyRemote(edit, invalidNewline); changed {
		t.Fatalf("expected changed=false for invalid newline mode")
	}
}

func TestRemapStatus_AllValuesReachable(t *testing.T) {
	statuses := []RemapStatus{
		RemapUnchanged,
		RemapMoved,
		RemapClamped,
		RemapInvalidated,
	}

	for _, status := range statuses {
		pt := RemapPoint{
			Before: Pos{Row: 1, GraphemeCol: 2},
			After:  Pos{Row: 3, GraphemeCol: 4},
			Status: status,
		}
		out := ApplyRemoteResult{
			Remap: RemapReport{
				Cursor:   pt,
				SelStart: pt,
				SelEnd:   pt,
			},
		}
		if out.Remap.Cursor.Status != status || out.Remap.SelStart.Status != status || out.Remap.SelEnd.Status != status {
			t.Fatalf("expected status=%v in all remap points", status)
		}
	}
}
