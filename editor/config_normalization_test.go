package editor

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestConfig_KeyMapZeroDefaults(t *testing.T) {
	m := New(Config{})
	if len(m.cfg.KeyMap.Left.Keys()) == 0 {
		t.Fatalf("expected default keymap to populate Left binding")
	}
}

func TestConfig_KeyMapPartialPreserved(t *testing.T) {
	km := KeyMap{
		Up: key.NewBinding(key.WithKeys("k")),
	}

	m := New(Config{KeyMap: km})
	if got := m.cfg.KeyMap.Up.Keys(); len(got) != 1 || got[0] != "k" {
		t.Fatalf("expected partial keymap Up binding to be preserved, got %v", got)
	}
	if got := m.cfg.KeyMap.Left.Keys(); len(got) != 0 {
		t.Fatalf("expected unspecified bindings to remain unset for partial keymap, got Left=%v", got)
	}
}

func TestConfig_CompletionKeyMapPartialPreserved(t *testing.T) {
	km := CompletionKeyMap{
		Accept: key.NewBinding(key.WithKeys("tab")),
	}

	m := New(Config{CompletionKeyMap: km})
	if got := m.cfg.CompletionKeyMap.Accept.Keys(); len(got) != 1 || got[0] != "tab" {
		t.Fatalf("expected partial completion keymap Accept binding to be preserved, got %v", got)
	}
	if got := m.cfg.CompletionKeyMap.Trigger.Keys(); len(got) != 0 {
		t.Fatalf("expected trigger to remain unset for partial completion keymap, got %v", got)
	}
	if m.cfg.CompletionKeyMap.AcceptTab {
		t.Fatalf("expected AcceptTab default to remain false for partial completion keymap")
	}
}

func TestConfig_ScrollbarMinThumbDefaultsToOne(t *testing.T) {
	m := New(Config{})
	if got, want := m.cfg.Scrollbar.MinThumb, 1; got != want {
		t.Fatalf("scrollbar min thumb default: got %d, want %d", got, want)
	}
}

func TestConfig_ScrollbarMinThumbClamp(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "zero", in: 0, want: 1},
		{name: "negative", in: -3, want: 1},
		{name: "positive", in: 4, want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(Config{Scrollbar: ScrollbarConfig{MinThumb: tt.in}})
			if got := m.cfg.Scrollbar.MinThumb; got != tt.want {
				t.Fatalf("min thumb normalize(%d): got %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
