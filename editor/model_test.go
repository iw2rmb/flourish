package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestModel_SetSizeAffectsViewHeight(t *testing.T) {
	m := New(Config{Text: "a\nb\nc"})
	m = m.Blur()

	m = m.SetSize(20, 2)
	if got := lipgloss.Height(m.View()); got != 2 {
		t.Fatalf("height after SetSize(20,2): got %d, want %d", got, 2)
	}

	m = m.SetSize(20, 4)
	if got := lipgloss.Height(m.View()); got != 4 {
		t.Fatalf("height after SetSize(20,4): got %d, want %d", got, 4)
	}
}

func TestView_SnapshotFixedSize(t *testing.T) {
	m := New(Config{
		Text:         "one\ntwo\nthree\nfour\nfive",
		ShowLineNums: true,
	})
	m = m.Blur()
	m = m.SetSize(8, 3)

	got := strings.Split(m.View(), "\n")
	if len(got) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(got))
	}
	for i := range got {
		got[i] = strings.TrimRight(got[i], " ")
	}

	want := []string{
		"1 one",
		"2 two",
		"3 three",
	}
	if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
		t.Fatalf("unexpected view:\n got: %q\nwant: %q", got, want)
	}
}
