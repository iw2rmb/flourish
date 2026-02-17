package editor

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
)

func (m Model) updateMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.cfg.ScrollPolicy == ScrollAllowManual || !isManualScrollMouse(msg) {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	if !m.focused || m.buf == nil {
		return m, cmd
	}

	// Only handle selection/cursor changes for left button interactions.
	switch msg.Action { //nolint:exhaustive
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, cmd
		}
		if !m.mouseInBounds(msg.X, msg.Y) {
			return m, cmd
		}

		p := m.screenToDocPos(msg.X, msg.Y)
		if msg.Shift {
			anchor := m.buf.Cursor()
			if raw, ok := m.buf.SelectionRaw(); ok {
				anchor = raw.Start
			}
			m.mouseAnchor = anchor
			m.buf.SetCursor(p)
			m.buf.SetSelection(buffer.Range{Start: anchor, End: p})
		} else {
			m.mouseAnchor = p
			m.buf.SetCursor(p)
			m.buf.ClearSelection()
		}
		m.mouseDragging = true

	case tea.MouseActionMotion:
		if !m.mouseDragging {
			return m, cmd
		}

		x, y := m.clampMouseToBounds(msg.X, msg.Y)
		p := m.screenToDocPos(x, y)
		m.buf.SetCursor(p)
		m.buf.SetSelection(buffer.Range{Start: m.mouseAnchor, End: p})

	case tea.MouseActionRelease:
		m.mouseDragging = false
	}

	return m, cmd
}

func isManualScrollMouse(msg tea.MouseMsg) bool {
	return msg.Action == tea.MouseActionPress &&
		(msg.Button == tea.MouseButtonWheelUp ||
			msg.Button == tea.MouseButtonWheelDown ||
			msg.Button == tea.MouseButtonWheelLeft ||
			msg.Button == tea.MouseButtonWheelRight)
}

func (m Model) mouseInBounds(x, y int) bool {
	if m.viewport.Width <= 0 || m.viewport.Height <= 0 {
		return false
	}
	return x >= 0 && x < m.viewport.Width && y >= 0 && y < m.viewport.Height
}

func (m Model) clampMouseToBounds(x, y int) (int, int) {
	if m.viewport.Width > 0 {
		if x < 0 {
			x = 0
		}
		if x >= m.viewport.Width {
			x = m.viewport.Width - 1
		}
	}
	if m.viewport.Height > 0 {
		if y < 0 {
			y = 0
		}
		if y >= m.viewport.Height {
			y = m.viewport.Height - 1
		}
	}
	return x, y
}
