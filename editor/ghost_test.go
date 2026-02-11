package editor

import (
	"testing"

	"github.com/iw2rmb/flouris/buffer"
)

func TestGhostProvider_CalledOnlyAtEOL_AndOnlyWhenFocused(t *testing.T) {
	calls := 0
	m := New(Config{
		Text: "ab",
		GhostProvider: func(ctx GhostContext) (Ghost, bool) {
			calls++
			return Ghost{Text: "X"}, true
		},
	})
	calls = 0 // New() triggers an initial render

	if _, ok := m.ghostForCursor(); ok {
		t.Fatalf("expected no ghost when cursor not at EOL")
	}
	if calls != 0 {
		t.Fatalf("provider calls when not at EOL: got %d, want %d", calls, 0)
	}

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost at EOL")
	}
	if calls != 1 {
		t.Fatalf("provider calls at EOL: got %d, want %d", calls, 1)
	}

	m = m.Blur()
	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // keep at EOL
	if _, ok := m.ghostForCursor(); ok {
		t.Fatalf("expected no ghost when blurred")
	}
	if calls != 1 {
		t.Fatalf("provider should not be called when blurred: got %d, want %d", calls, 1)
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

	m.buf.SetCursor(buffer.Pos{Row: 0, GraphemeCol: 2}) // EOL

	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost on first call")
	}
	if _, ok := m.ghostForCursor(); !ok {
		t.Fatalf("expected ghost on cached call")
	}
	if calls != 1 {
		t.Fatalf("provider calls with cache: got %d, want %d", calls, 1)
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
