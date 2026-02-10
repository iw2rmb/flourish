package editor

// WrapMode controls how long logical lines are displayed.
//
// WrapNone renders one logical line per visual row and uses horizontal scrolling
// to keep the cursor visible. Other wrap modes are implemented in later phases.
type WrapMode int

const (
	WrapNone WrapMode = iota
	WrapWord
	WrapGrapheme
)

