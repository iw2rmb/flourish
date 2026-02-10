package editor

// Config configures the editor Model.
//
// Phase 6 adds key handling, selection rendering, and scroll-follow behavior.
type Config struct {
	// Initial text for the internal buffer.
	Text string
	// Optional host metadata used by hooks for caching.
	DocID string

	// Rendering options.
	ShowLineNums bool
	Style        Style
	// WrapMode controls soft wrapping vs horizontal scrolling. Default is WrapNone.
	WrapMode WrapMode

	// VirtualTextProvider optionally supplies per-line view-only transforms
	// (virtual deletions/insertions) used by the VisualLine mapping layer.
	// When nil, the transform is identity.
	VirtualTextProvider VirtualTextProvider

	// TabWidth controls tab stop width in terminal cells. If <= 0, defaults to 4.
	TabWidth int

	// If true, movement/selection still work but buffer mutations are ignored.
	ReadOnly bool

	// KeyMap controls default keybindings. Zero value uses DefaultKeyMap().
	KeyMap KeyMap

	// Clipboard is optional. If nil, copy/cut/paste are disabled.
	Clipboard Clipboard

	// Forwarded to buffer.Options.
	HistoryLimit int

	// Ghost suggestion (EOL-only). When nil, ghost is disabled.
	GhostProvider GhostProvider
	// GhostAccept configures accept keys. Zero value defaults to Tab+Right.
	GhostAccept GhostAccept

	// Highlighter optionally provides per-line highlight spans over the visible
	// text after virtual deletions.
	Highlighter Highlighter

	// OnChange, if set, fires after every successful buffer mutation triggered
	// via Update. It is not fired for host-driven buffer changes.
	OnChange func(ChangeEvent)
}
