package editor

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStyleIsZero_ScrollbarFieldsAffectZeroCheck(t *testing.T) {
	tests := []struct {
		name  string
		style Style
	}{
		{
			name:  "track",
			style: Style{ScrollbarTrack: lipgloss.NewStyle().PaddingLeft(1)},
		},
		{
			name:  "thumb",
			style: Style{ScrollbarThumb: lipgloss.NewStyle().PaddingLeft(1)},
		},
		{
			name:  "corner",
			style: Style{ScrollbarCorner: lipgloss.NewStyle().PaddingLeft(1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.style.isZero() {
				t.Fatalf("expected style with %s field set to be non-zero", tt.name)
			}
		})
	}
}

func TestDefaultStyle_ScrollbarStylesAreConfigured(t *testing.T) {
	st := DefaultStyle()

	if st.ScrollbarTrack.GetBackground() == nil {
		t.Fatalf("expected default ScrollbarTrack background to be configured")
	}
	if st.ScrollbarThumb.GetBackground() == nil {
		t.Fatalf("expected default ScrollbarThumb background to be configured")
	}
	if st.ScrollbarCorner.GetBackground() == nil {
		t.Fatalf("expected default ScrollbarCorner background to be configured")
	}
}
