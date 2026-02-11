package grapheme

import "testing"

func TestSplitAndCount_MultiRuneGraphemes(t *testing.T) {
	text := "a" + "e\u0301" + "👨‍👩‍👧‍👦" + "b"
	got := Split(text)
	if len(got) != 4 {
		t.Fatalf("split len=%d, want %d", len(got), 4)
	}
	if got[1] != "e\u0301" {
		t.Fatalf("split[1]=%q, want %q", got[1], "e\u0301")
	}
	if got[2] != "👨‍👩‍👧‍👦" {
		t.Fatalf("split[2]=%q, want family emoji", got[2])
	}
	if c := Count(text); c != 4 {
		t.Fatalf("count=%d, want %d", c, 4)
	}
}

func TestSlice_GraphemeSafe(t *testing.T) {
	text := "a" + "e\u0301" + "👨‍👩‍👧‍👦" + "b"
	if got, want := Slice(text, 1, 3), "e\u0301👨‍👩‍👧‍👦"; got != want {
		t.Fatalf("slice=%q, want %q", got, want)
	}
	if got := Slice(text, 5, 6); got != "" {
		t.Fatalf("slice past end=%q, want empty", got)
	}
}

func TestClassifiers(t *testing.T) {
	if !IsSpace("\t") {
		t.Fatalf("tab should be space")
	}
	if IsSpace("a") {
		t.Fatalf("letter should not be space")
	}
	if !IsPunct("!") {
		t.Fatalf("exclamation should be punct")
	}
	if IsPunct("a") {
		t.Fatalf("letter should not be punct")
	}
}
