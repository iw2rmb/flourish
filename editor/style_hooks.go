package editor

// RowStyleContext describes a rendered visual row for row-level style overrides.
type RowStyleContext struct {
	// Row is the logical document row index.
	Row int
	// SegmentIndex is the wrapped visual segment index within Row.
	SegmentIndex int

	// RawText is the unwrapped buffer line text.
	RawText string
	// Text is the visible line text after virtual deletions.
	Text string

	// IsActive reports whether Row is the cursor row.
	IsActive bool
	// IsFocused reports whether the editor is focused.
	IsFocused bool
}

// TokenStyleContext describes a rendered visual token for token-level style overrides.
type TokenStyleContext struct {
	// Row is the logical document row index.
	Row int
	// SegmentIndex is the wrapped visual segment index within Row.
	SegmentIndex int

	// Token is the current rendered visual token.
	Token VisualToken

	// RawText is the unwrapped buffer line text.
	RawText string
	// Text is the visible line text after virtual deletions.
	Text string

	// IsActiveRow reports whether Row is the cursor row.
	IsActiveRow bool
	// IsFocused reports whether the editor is focused.
	IsFocused bool

	// IsHighlighted reports overlap with highlighter spans.
	IsHighlighted bool
	// IsSelected reports overlap with the active document selection range.
	IsSelected bool

	// IsLink reports whether link style/target are active for this token.
	IsLink bool
	// LinkTarget is set when IsLink is true.
	LinkTarget string
}
