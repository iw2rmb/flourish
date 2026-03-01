package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// compositeTopLeft overlays fg onto bg using top-left anchoring with explicit offsets.
func compositeTopLeft(fg, bg string, xOff, yOff int) string {
	if fg == "" {
		return bg
	}
	if bg == "" {
		return fg
	}
	if strings.Count(fg, "\n") == 0 && strings.Count(bg, "\n") == 0 {
		return fg
	}

	fgWidth, fgHeight := lipgloss.Size(fg)
	bgWidth, bgHeight := lipgloss.Size(bg)
	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		return fg
	}

	x := clampInt(xOff, 0, bgWidth-fgWidth)
	y := clampInt(yOff, 0, bgHeight-fgHeight)
	base := lipgloss.NewLayer(bg)
	overlay := lipgloss.NewLayer(fg).X(x).Y(y).Z(1)
	return lipgloss.NewCompositor(base, overlay).Render()
}
