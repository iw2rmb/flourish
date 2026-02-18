package buffer

import "testing"

func TestBuffer_ApplyRemote_APIAndRemoteChange(t *testing.T) {
	b := New("hello", Options{})
	b.SetCursor(Pos{Row: 0, GraphemeCol: 2})
	v := b.Version()

	opts := ApplyRemoteOptions{
		BaseVersion: b.Version(),
		ClampPolicy: ConvertPolicy{
			ClampMode:   OffsetClamp,
			NewlineMode: NewlineAsSingleRune,
		},
	}

	res, changed := b.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 0},
			},
			Text: "X",
			OpID: "op-1",
		},
	}, opts)
	if !changed {
		t.Fatalf("expected changed=true")
	}

	if got, want := b.Text(), "Xhello"; got != want {
		t.Fatalf("text=%q, want %q", got, want)
	}
	if got, want := b.Version(), v+1; got != want {
		t.Fatalf("version=%d, want %d", got, want)
	}
	if got, want := b.Cursor(), (Pos{Row: 0, GraphemeCol: 2}); got != want {
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
	if got, want := res.Remap.Cursor.After, (Pos{Row: 0, GraphemeCol: 2}); got != want {
		t.Fatalf("cursor after=%v, want %v", got, want)
	}
	if got, want := res.Remap.Cursor.Status, RemapUnchanged; got != want {
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
	}, ApplyRemoteOptions{})
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
	b.SetCursor(Pos{Row: 0, GraphemeCol: 3})

	res, changed := b.ApplyRemote([]RemoteEdit{
		{
			Range: Range{
				Start: Pos{Row: 0, GraphemeCol: 0},
				End:   Pos{Row: 0, GraphemeCol: 3},
			},
			Text: "",
		},
	}, ApplyRemoteOptions{})
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
	}, ApplyRemoteOptions{})
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
