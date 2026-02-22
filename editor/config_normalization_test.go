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
