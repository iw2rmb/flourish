package editor

// WrapMode controls how long logical lines are displayed.
//
// WrapNone renders one logical line per visual row and uses horizontal scrolling
// to keep the cursor visible. WrapWord and WrapGrapheme use soft wrapping.
type WrapMode int

const (
	WrapNone WrapMode = iota
	WrapWord
	WrapGrapheme
)
