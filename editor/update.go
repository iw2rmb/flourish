package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flouris/buffer"
)

func (m Model) updateKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if !m.focused || m.buf == nil {
		return m, nil
	}

	// Paste events should always insert literal text and never trigger shortcuts.
	if msg.Type == tea.KeyRunes && msg.Paste && len(msg.Runes) > 0 {
		if !m.cfg.ReadOnly {
			m.buf.InsertText(string(msg.Runes))
		}
		return m, nil
	}

	km := m.cfg.KeyMap
	ga := normalizeGhostAccept(m.cfg.GhostAccept)

	if m.cfg.GhostProvider != nil && !m.cfg.ReadOnly {
		if ga.AcceptTab && msg.Type == tea.KeyTab {
			if ghost, ok := (&m).ghostForCursor(); ok && len(ghost.Edits) > 0 {
				m.buf.Apply(ghost.Edits...)
				return m, nil
			}
		}
		if ga.AcceptRight && key.Matches(msg, km.Right) {
			if ghost, ok := (&m).ghostForCursor(); ok && len(ghost.Edits) > 0 {
				m.buf.Apply(ghost.Edits...)
				return m, nil
			}
		}
	}

	switch {
	case key.Matches(msg, km.Left):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirLeft})
	case key.Matches(msg, km.Right):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirRight})
	case key.Matches(msg, km.Up):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirUp})
	case key.Matches(msg, km.Down):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirDown})

	case key.Matches(msg, km.ShiftLeft):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirLeft, Extend: true})
	case key.Matches(msg, km.ShiftRight):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirRight, Extend: true})
	case key.Matches(msg, km.ShiftUp):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirUp, Extend: true})
	case key.Matches(msg, km.ShiftDown):
		m.buf.Move(buffer.Move{Unit: buffer.MoveRune, Dir: buffer.DirDown, Extend: true})

	case key.Matches(msg, km.WordLeft):
		m.buf.Move(buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirLeft})
	case key.Matches(msg, km.WordRight):
		m.buf.Move(buffer.Move{Unit: buffer.MoveWord, Dir: buffer.DirRight})

	case key.Matches(msg, km.Home):
		m.buf.Move(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirHome})
	case key.Matches(msg, km.End):
		m.buf.Move(buffer.Move{Unit: buffer.MoveLine, Dir: buffer.DirEnd})

	case key.Matches(msg, km.Backspace):
		if !m.cfg.ReadOnly {
			m.buf.DeleteBackward()
		}
	case key.Matches(msg, km.Delete):
		if !m.cfg.ReadOnly {
			m.buf.DeleteForward()
		}
	case key.Matches(msg, km.Enter):
		if !m.cfg.ReadOnly {
			m.buf.InsertNewline()
		}

	case key.Matches(msg, km.Undo):
		if !m.cfg.ReadOnly {
			_ = m.buf.Undo()
		}
	case key.Matches(msg, km.Redo):
		if !m.cfg.ReadOnly {
			_ = m.buf.Redo()
		}

	case key.Matches(msg, km.Copy):
		m.copySelection()
	case key.Matches(msg, km.Cut):
		if !m.cfg.ReadOnly {
			m.cutSelection()
		} else {
			m.copySelection()
		}
	case key.Matches(msg, km.Paste):
		if !m.cfg.ReadOnly {
			m.pasteClipboard()
		}

	default:
		if msg.Type == tea.KeyTab {
			if !m.cfg.ReadOnly {
				m.buf.InsertRune('\t')
			}
			return m, nil
		}

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && !msg.Alt {
			if !m.cfg.ReadOnly {
				m.buf.InsertText(string(msg.Runes))
			}
		}
	}

	return m, nil
}

func (m Model) copySelection() {
	if m.cfg.Clipboard == nil || m.buf == nil {
		return
	}
	r, ok := m.buf.Selection()
	if !ok {
		return
	}
	s := textInRange(m.buf.Text(), r)
	if s == "" {
		return
	}
	_ = m.cfg.Clipboard.WriteText(s)
}

func (m Model) cutSelection() {
	if m.cfg.Clipboard == nil || m.buf == nil {
		return
	}
	r, ok := m.buf.Selection()
	if !ok {
		return
	}
	s := textInRange(m.buf.Text(), r)
	if s != "" {
		_ = m.cfg.Clipboard.WriteText(s)
	}
	m.buf.DeleteSelection()
}

func (m Model) pasteClipboard() {
	if m.cfg.Clipboard == nil || m.buf == nil {
		return
	}
	s, err := m.cfg.Clipboard.ReadText()
	if err != nil || s == "" {
		return
	}
	// Normalize newlines from external sources.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	m.buf.InsertText(s)
}

func textInRange(text string, r buffer.Range) string {
	r = buffer.NormalizeRange(r)
	if r.IsEmpty() {
		return ""
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}

	if r.Start.Row < 0 || r.Start.Row >= len(lines) || r.End.Row < 0 || r.End.Row >= len(lines) {
		return ""
	}

	if r.Start.Row == r.End.Row {
		rr := []rune(lines[r.Start.Row])
		if r.Start.Col < 0 || r.End.Col < 0 || r.Start.Col > len(rr) || r.End.Col > len(rr) {
			return ""
		}
		return string(rr[r.Start.Col:r.End.Col])
	}

	var sb strings.Builder
	for row := r.Start.Row; row <= r.End.Row; row++ {
		if row > r.Start.Row {
			sb.WriteByte('\n')
		}
		rr := []rune(lines[row])
		startCol := 0
		endCol := len(rr)
		if row == r.Start.Row {
			startCol = r.Start.Col
		}
		if row == r.End.Row {
			endCol = r.End.Col
		}
		if startCol < 0 || endCol < 0 || startCol > len(rr) || endCol > len(rr) || startCol > endCol {
			return ""
		}
		sb.WriteString(string(rr[startCol:endCol]))
	}
	return sb.String()
}
