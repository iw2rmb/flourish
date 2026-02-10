package editor

// Config configures the editor Model.
//
// Phase 6 adds key handling, selection rendering, and scroll-follow behavior.
type Config struct {
	// Initial text for the internal buffer.
	Text string

	// Rendering options.
	ShowLineNums bool
	Style        Style

	// If true, movement/selection still work but buffer mutations are ignored.
	ReadOnly bool

	// KeyMap controls default keybindings. Zero value uses DefaultKeyMap().
	KeyMap KeyMap

	// Clipboard is optional. If nil, copy/cut/paste are disabled.
	Clipboard Clipboard

	// Forwarded to buffer.Options.
	HistoryLimit int
}
