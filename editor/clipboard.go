package editor

// Clipboard provides editor-level clipboard integration.
//
// Errors must not crash the UI; failures are ignored (v0).
type Clipboard interface {
	ReadText() (string, error)
	WriteText(s string) error
}
