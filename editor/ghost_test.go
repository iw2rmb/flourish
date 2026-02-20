package editor

import (
	"testing"

	"github.com/iw2rmb/flourish/buffer"
)

func TestGhostProvider_CalledAtAnyCol_AndOnlyWhenFocused(t *testing.T) {
	calls := 0
	var seen []GhostContext
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			calls++
			seen = append(seen, ctx)
			return Ghost{Text: "X"}, true
		},
	})
	calls = 0 // New() triggers an initial render
	seen = nil

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1}) // non-EOL
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost at non-EOL")
	}
	if calls != 1 {
		t.Fatalf("provider calls at non-EOL: got %d, want %d", calls, 1)
	}
	if len(seen) != 1 || seen[0].GraphemeCol != 1 || seen[0].IsEndOfLine {
		t.Fatalf("non-EOL context: got %+v", seen)
	}

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost at EOL")
	}
	if calls != 2 {
		t.Fatalf("provider calls at EOL: got %d, want %d", calls, 2)
	}
	if len(seen) != 2 || seen[1].GraphemeCol != 2 || !seen[1].IsEndOfLine {
		t.Fatalf("EOL context: got %+v", seen)
	}

	m = m.Blur()
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1})
	if _, ok := m.ghostForCursor(); ok {
		t.Fatalf("expected no ghost when blurred")
	}
	if calls != 2 {
		t.Fatalf("provider should not be called when blurred: got %d, want %d", calls, 2)
	}
}

func TestGhostProvider_CacheHitAvoidsDuplicateCalls(t *testing.T) {
	calls := 0
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			calls++
			return Ghost{Text: "X"}, true
		},
	})
	calls = 0

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1}) // non-EOL

	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost on first call")
	}
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost on cached call")
	}
	if calls != 1 {
		t.Fatalf("provider calls with cache: got %d, want %d", calls, 1)
	}

	// Cursor column is part of the key.
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost at a different cursor col")
	}
	if calls != 2 {
		t.Fatalf("provider calls after cursor-col change: got %d, want %d", calls, 2)
	}
}

func TestGhostProvider_ContextIncludesDocID_AndCacheKeysByIt(t *testing.T) {
	var seenDocIDs []string
	calls := 0
	m := New(Config{
		Text:  "ab",
		DocID: "doc-a",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			calls++
			seenDocIDs = append(seenDocIDs, ctx.DocID)
			return Ghost{Text: "X"}, true
		},
	})
	calls = 0
	seenDocIDs = nil

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL

	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost on first call")
	}
	if calls != 1 {
		t.Fatalf("provider calls: got %d, want %d", calls, 1)
	}
	if len(seenDocIDs) != 1 || seenDocIDs[0] != "doc-a" {
		t.Fatalf("seen doc IDs: got %v, want %v", seenDocIDs, []string{"doc-a"})
	}

	// Cache hit for same doc.
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost on cached call")
	}
	if calls != 1 {
		t.Fatalf("provider calls with cache: got %d, want %d", calls, 1)
	}

	// Doc switch invalidates cache when DocID metadata is provided.
	m.cfg.DocID = "doc-b"
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost on doc switch")
	}
	if calls != 2 {
		t.Fatalf("provider calls after doc switch: got %d, want %d", calls, 2)
	}
}

func TestGhostProvider_SuppressedWhileCompletionVisible(t *testing.T) {
	calls := 0
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			calls++
			return Ghost{Text: "X"}, true
		},
	})
	calls = 0
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 1})

	m = m.SetCompletionState(CompletionState{
		Visible: true,
		Items: []CompletionItem{
			{ID: "0", InsertText: "X"},
		},
		VisibleIndices: []int{0},
	})
	if _, ok := m.ghostForCursor(); ok {
		t.Fatalf("ghost should be suppressed while completion popup is visible")
	}
	if got, want := calls, 0; got != want {
		t.Fatalf("provider calls while completion visible: got %d, want %d", got, want)
	}

	m = m.ClearCompletion()
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("ghost should return after completion is cleared")
	}
	if got, want := calls, 1; got != want {
		t.Fatalf("provider calls after completion cleared: got %d, want %d", got, want)
	}
}
