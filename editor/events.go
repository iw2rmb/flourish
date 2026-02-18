package editor

import "github.com/iw2rmb/flourish/buffer"

type ChangeEvent struct {
	Change buffer.Change
}

func buildChangeEvent(b *buffer.Buffer) (ChangeEvent, bool) {
	change, ok := b.LastChange()
	if !ok {
		return ChangeEvent{}, false
	}
	return ChangeEvent{Change: change}, true
}
