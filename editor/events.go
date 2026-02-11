package editor

import "github.com/iw2rmb/flouris/buffer"

type ChangeEvent struct {
	Version   uint64
	Cursor    buffer.Pos
	Selection struct {
		Range  buffer.Range
		Active bool
	}

	// v0: simplest payload; host can diff if needed.
	Text string
}

func buildChangeEvent(b *buffer.Buffer) ChangeEvent {
	ev := ChangeEvent{
		Version: b.Version(),
		Cursor:  b.Cursor(),
		Text:    b.Text(),
	}
	if r, ok := b.Selection(); ok {
		ev.Selection.Active = true
		ev.Selection.Range = r
	}
	return ev
}
