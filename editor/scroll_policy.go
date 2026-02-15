package editor

// ScrollPolicy controls how viewport scrolling is allowed to move relative to
// the cursor.
type ScrollPolicy int

const (
	// ScrollAllowManual allows manual viewport scrolling (for example via mouse
	// wheel) even when the cursor does not move.
	ScrollAllowManual ScrollPolicy = iota
	// ScrollFollowCursorOnly keeps vertical viewport movement cursor-driven.
	// Manual viewport scrolling is ignored.
	ScrollFollowCursorOnly
)
