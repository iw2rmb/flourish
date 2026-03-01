package editor

import (
	tea "charm.land/bubbletea/v2"

	"github.com/iw2rmb/flourish/buffer"
)

func (m Model) updateMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	if handled := m.handleScrollbarMouse(msg, m.cfg.ScrollPolicy == ScrollAllowManual); handled {
		return m, nil
	}
	mouse := msg.Mouse()

	var cmd tea.Cmd
	if m.cfg.ScrollPolicy == ScrollAllowManual || !isManualScrollMouse(msg) {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	if !m.focused || m.buf == nil {
		if _, ok := msg.(tea.MouseReleaseMsg); ok {
			m.mouseDragging = false
		}
		return m, cmd
	}

	// Only handle selection/cursor changes for left button interactions.
	switch msg := msg.(type) { //nolint:exhaustive
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, cmd
		}
		if !m.mouseInBounds(msg.X, msg.Y) {
			return m, cmd
		}

		p := m.screenToDocPos(msg.X, msg.Y)
		if msg.Mod&tea.ModShift != 0 {
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

	case tea.MouseMotionMsg:
		if !m.mouseDragging {
			return m, cmd
		}

		x, y := m.clampMouseToBounds(msg.X, msg.Y)
		p := m.screenToDocPos(x, y)
		m.buf.SetCursor(p)
		m.buf.SetSelection(buffer.Range{Start: m.mouseAnchor, End: p})

	case tea.MouseReleaseMsg:
		_ = mouse
		m.mouseDragging = false
		m.clearScrollbarDrag()
	}

	return m, cmd
}

type scrollbarHitPart int

const (
	scrollbarHitNone scrollbarHitPart = iota
	scrollbarHitTrackBeforeThumb
	scrollbarHitThumb
	scrollbarHitTrackAfterThumb
)

type scrollbarHit struct {
	axis scrollbarDragAxis
	part scrollbarHitPart
	cell int
}

func (m *Model) handleScrollbarMouse(msg tea.MouseMsg, allowManual bool) bool {
	if m.buf == nil {
		return false
	}

	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	metrics := m.resolveScrollbarMetrics(lines, layout)
	gutterWidth := m.resolvedGutterWidth(len(lines))

	mouse := msg.Mouse()
	switch msg := msg.(type) { //nolint:exhaustive
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft || !m.mouseInBounds(msg.X, msg.Y) {
			return false
		}
		hit, ok := resolveScrollbarHit(msg.X, msg.Y, metrics, gutterWidth)
		if !ok {
			return false
		}
		if !allowManual {
			return true
		}
		m.applyScrollbarPress(hit, metrics)
		return true

	case tea.MouseMotionMsg:
		if m.scrollbarDragAxis == dragNone {
			return false
		}
		if !allowManual {
			m.clearScrollbarDrag()
			return true
		}

		x, y := m.clampMouseToBounds(msg.X, msg.Y)
		switch m.scrollbarDragAxis {
		case dragVertical:
			cell := y
			m.applyVerticalScrollbarDrag(cell, metrics)
		case dragHorizontal:
			left := gutterWidth
			right := left + metrics.contentWidth - 1
			cell := clampInt(x-left, 0, max(right-left, 0))
			m.applyHorizontalScrollbarDrag(cell, metrics)
		}
		return true

	case tea.MouseReleaseMsg:
		_ = mouse
		if m.scrollbarDragAxis != dragNone {
			m.clearScrollbarDrag()
			return true
		}
		return false
	}

	return false
}

func resolveScrollbarHit(x, y int, metrics scrollbarMetrics, gutterWidth int) (scrollbarHit, bool) {
	if metrics.innerWidth <= 0 || metrics.innerHeight <= 0 {
		return scrollbarHit{}, false
	}

	if metrics.showV {
		vX := metrics.innerWidth - 1
		if x == vX && y >= 0 && y < metrics.contentHeight {
			return scrollbarHit{
				axis: dragVertical,
				part: classifyScrollbarPart(y, metrics.vThumbPos, metrics.vThumbLen),
				cell: y,
			}, true
		}
	}

	if metrics.showH {
		hY := metrics.innerHeight - 1
		startX := gutterWidth
		endX := startX + metrics.contentWidth
		if y == hY && x >= startX && x < endX {
			cell := x - startX
			return scrollbarHit{
				axis: dragHorizontal,
				part: classifyScrollbarPart(cell, metrics.hThumbPos, metrics.hThumbLen),
				cell: cell,
			}, true
		}
	}

	return scrollbarHit{}, false
}

func classifyScrollbarPart(cell, thumbPos, thumbLen int) scrollbarHitPart {
	thumbStart := max(thumbPos, 0)
	thumbEnd := thumbStart + max(thumbLen, 0)

	if cell >= thumbStart && cell < thumbEnd {
		return scrollbarHitThumb
	}
	if cell < thumbStart {
		return scrollbarHitTrackBeforeThumb
	}
	return scrollbarHitTrackAfterThumb
}

func (m *Model) applyScrollbarPress(hit scrollbarHit, metrics scrollbarMetrics) {
	switch hit.axis {
	case dragVertical:
		switch hit.part {
		case scrollbarHitThumb:
			m.startScrollbarDrag(dragVertical, hit.cell, metrics.yOffset)
		case scrollbarHitTrackBeforeThumb:
			m.pageVerticalScrollbar(-metrics.contentHeight, metrics)
		case scrollbarHitTrackAfterThumb:
			m.pageVerticalScrollbar(metrics.contentHeight, metrics)
		}
	case dragHorizontal:
		switch hit.part {
		case scrollbarHitThumb:
			m.startScrollbarDrag(dragHorizontal, hit.cell, metrics.xOffset)
		case scrollbarHitTrackBeforeThumb:
			m.pageHorizontalScrollbar(-metrics.contentWidth, metrics)
		case scrollbarHitTrackAfterThumb:
			m.pageHorizontalScrollbar(metrics.contentWidth, metrics)
		}
	}
}

func (m *Model) startScrollbarDrag(axis scrollbarDragAxis, startCell, startOffset int) {
	m.scrollbarDragAxis = axis
	m.scrollbarDragStartCell = max(startCell, 0)
	m.scrollbarDragStartOffset = max(startOffset, 0)
}

func (m *Model) clearScrollbarDrag() {
	m.scrollbarDragAxis = dragNone
	m.scrollbarDragStartCell = 0
	m.scrollbarDragStartOffset = 0
}

func (m *Model) pageVerticalScrollbar(delta int, metrics scrollbarMetrics) {
	maxYOffset := max(metrics.totalRows-metrics.contentHeight, 0)
	next := clampInt(metrics.yOffset+delta, 0, maxYOffset)
	if next != m.viewport.YOffset() {
		m.viewport.SetYOffset(next)
	}
}

func (m *Model) pageHorizontalScrollbar(delta int, metrics scrollbarMetrics) {
	maxXOffset := max(metrics.totalCols-metrics.contentWidth, 0)
	next := clampInt(metrics.xOffset+delta, 0, maxXOffset)
	if next != m.xOffset {
		m.xOffset = next
		m.rebuildContent()
	}
}

func (m *Model) applyVerticalScrollbarDrag(pointerCell int, metrics scrollbarMetrics) {
	maxYOffset := max(metrics.totalRows-metrics.contentHeight, 0)
	next := scrollbarDragOffset(
		m.scrollbarDragStartOffset,
		pointerCell-m.scrollbarDragStartCell,
		metrics.contentHeight,
		metrics.vThumbLen,
		maxYOffset,
	)
	next = clampInt(next, 0, maxYOffset)
	if next != m.viewport.YOffset() {
		m.viewport.SetYOffset(next)
	}
}

func (m *Model) applyHorizontalScrollbarDrag(pointerCell int, metrics scrollbarMetrics) {
	maxXOffset := max(metrics.totalCols-metrics.contentWidth, 0)
	next := scrollbarDragOffset(
		m.scrollbarDragStartOffset,
		pointerCell-m.scrollbarDragStartCell,
		metrics.contentWidth,
		metrics.hThumbLen,
		maxXOffset,
	)
	next = clampInt(next, 0, maxXOffset)
	if next != m.xOffset {
		m.xOffset = next
		m.rebuildContent()
	}
}

func scrollbarDragOffset(startOffset, deltaCells, trackLen, thumbLen, maxOffset int) int {
	if maxOffset <= 0 {
		return 0
	}
	trackRange := trackLen - thumbLen
	if trackRange <= 0 {
		return clampInt(startOffset, 0, maxOffset)
	}
	deltaOffset := roundDiv(deltaCells*maxOffset, trackRange)
	return startOffset + deltaOffset
}

func roundDiv(n, d int) int {
	if d == 0 {
		return 0
	}
	if n >= 0 {
		return (n + d/2) / d
	}
	return -((-n + d/2) / d)
}

func isManualScrollMouse(msg tea.MouseMsg) bool {
	wheel, ok := msg.(tea.MouseWheelMsg)
	if !ok {
		return false
	}
	return wheel.Button == tea.MouseWheelUp ||
		wheel.Button == tea.MouseWheelDown ||
		wheel.Button == tea.MouseWheelLeft ||
		wheel.Button == tea.MouseWheelRight
}

func (m Model) mouseInBounds(x, y int) bool {
	if m.viewport.Width() <= 0 || m.viewport.Height() <= 0 {
		return false
	}
	return x >= 0 && x < m.viewport.Width() && y >= 0 && y < m.viewport.Height()
}

func (m Model) clampMouseToBounds(x, y int) (int, int) {
	if m.viewport.Width() > 0 {
		if x < 0 {
			x = 0
		}
		if x >= m.viewport.Width() {
			x = m.viewport.Width() - 1
		}
	}
	if m.viewport.Height() > 0 {
		if y < 0 {
			y = 0
		}
		if y >= m.viewport.Height() {
			y = m.viewport.Height() - 1
		}
	}
	return x, y
}
