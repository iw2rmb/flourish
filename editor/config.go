package editor

// Config configures the editor Model.
//
// Phase 5 uses only a small subset of the planned API.
type Config struct {
	// Initial text for the internal buffer.
	Text string

	// Rendering options.
	ShowLineNums bool
	Style        Style

	// Forwarded to buffer.Options.
	HistoryLimit int
}
