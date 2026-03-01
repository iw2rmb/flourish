package editor

import (
	"io"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

func init() {
	lipgloss.Writer = &colorprofile.Writer{
		Forward: io.Discard,
		Profile: colorprofile.TrueColor,
	}
}
