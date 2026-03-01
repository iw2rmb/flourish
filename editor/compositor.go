package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

	fgLines := compositorLines(fg)
	bgLines := compositorLines(bg)
	var sb strings.Builder

	for i, bgLine := range bgLines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i < y || i >= y+fgHeight {
			sb.WriteString(bgLine)
			continue
		}

		pos := 0
		if x > 0 {
			left := ansi.Truncate(bgLine, x, "")
			pos = ansi.StringWidth(left)
			sb.WriteString(left)
			if pos < x {
				sb.WriteString(spaceString(x - pos))
				pos = x
			}
		}

		fgLine := fgLines[i-y]
		sb.WriteString(fgLine)
		pos += ansi.StringWidth(fgLine)

		right := ansi.TruncateLeft(bgLine, pos, "")
		bgLineWidth := ansi.StringWidth(bgLine)
		rightWidth := ansi.StringWidth(right)
		if rightWidth <= bgLineWidth-pos {
			sb.WriteString(spaceString(bgLineWidth - rightWidth - pos))
		}
		sb.WriteString(right)
	}
	return sb.String()
}

func compositorLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}
