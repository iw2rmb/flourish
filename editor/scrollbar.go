package editor

type scrollbarMetrics struct {
	showV bool
	showH bool

	innerWidth  int
	innerHeight int

	contentWidth  int
	contentHeight int

	totalRows int
	yOffset   int
	vThumbPos int
	vThumbLen int

	totalCols int
	xOffset   int
	hThumbPos int
	hThumbLen int
}

func (m *Model) resolveScrollbarMetrics(lines []string, layout wrapLayoutCache) scrollbarMetrics {
	metrics := scrollbarMetrics{
		innerWidth:  m.viewport.Width - m.viewport.Style.GetHorizontalFrameSize(),
		innerHeight: m.viewport.Height - m.viewport.Style.GetVerticalFrameSize(),
	}
	if metrics.innerWidth < 0 {
		metrics.innerWidth = 0
	}
	if metrics.innerHeight < 0 {
		metrics.innerHeight = 0
	}

	gutterWidth := m.resolvedGutterWidth(len(lines))
	baseContentWidth := metrics.innerWidth - gutterWidth
	if baseContentWidth < 0 {
		baseContentWidth = 0
	}

	showV := m.cfg.Scrollbar.Vertical == ScrollbarAlways
	showH := m.cfg.WrapMode == WrapNone && m.cfg.Scrollbar.Horizontal == ScrollbarAlways
	stable := false

	for i := 0; i < 4; i++ {
		contentWidth := baseContentWidth
		if showV {
			contentWidth = max(contentWidth-1, 0)
		}
		contentHeight := metrics.innerHeight
		if showH {
			contentHeight = max(contentHeight-1, 0)
		}

		totalRows, totalCols := m.measureScrollExtents(lines, layout, contentWidth)
		nextShowV := resolveScrollbarAxisVisibility(
			m.cfg.Scrollbar.Vertical,
			totalRows,
			contentHeight,
			baseContentWidth > 1 && metrics.innerHeight > 0,
		)
		nextShowH := false
		if m.cfg.WrapMode == WrapNone {
			nextShowH = resolveScrollbarAxisVisibility(
				m.cfg.Scrollbar.Horizontal,
				totalCols,
				contentWidth,
				metrics.innerHeight > 1 && baseContentWidth > 0,
			)
		}

		metrics.showV = showV
		metrics.showH = showH
		metrics.contentWidth = contentWidth
		metrics.contentHeight = contentHeight
		metrics.totalRows = totalRows
		metrics.totalCols = totalCols

		if nextShowV == showV && nextShowH == showH {
			stable = true
			break
		}
		showV = nextShowV
		showH = nextShowH
	}

	if !stable {
		contentWidth := baseContentWidth
		if showV {
			contentWidth = max(contentWidth-1, 0)
		}
		contentHeight := metrics.innerHeight
		if showH {
			contentHeight = max(contentHeight-1, 0)
		}
		totalRows, totalCols := m.measureScrollExtents(lines, layout, contentWidth)
		metrics.showV = showV
		metrics.showH = showH
		metrics.contentWidth = contentWidth
		metrics.contentHeight = contentHeight
		metrics.totalRows = totalRows
		metrics.totalCols = totalCols
	}

	maxYOffset := 0
	if metrics.contentHeight > 0 && metrics.totalRows > metrics.contentHeight {
		maxYOffset = metrics.totalRows - metrics.contentHeight
	}
	metrics.yOffset = clampInt(m.viewport.YOffset, 0, maxYOffset)

	metrics.xOffset = 0
	if m.cfg.WrapMode == WrapNone {
		maxXOffset := 0
		if metrics.contentWidth > 0 && metrics.totalCols > metrics.contentWidth {
			maxXOffset = metrics.totalCols - metrics.contentWidth
		}
		metrics.xOffset = clampInt(m.xOffset, 0, maxXOffset)
	}

	if metrics.showV {
		metrics.vThumbPos, metrics.vThumbLen = resolveScrollbarThumb(
			metrics.contentHeight,
			metrics.contentHeight,
			metrics.totalRows,
			metrics.yOffset,
			m.cfg.Scrollbar.MinThumb,
		)
	}
	if metrics.showH && m.cfg.WrapMode == WrapNone {
		metrics.hThumbPos, metrics.hThumbLen = resolveScrollbarThumb(
			metrics.contentWidth,
			metrics.contentWidth,
			metrics.totalCols,
			metrics.xOffset,
			m.cfg.Scrollbar.MinThumb,
		)
	}

	return metrics
}

func resolveScrollbarAxisVisibility(mode ScrollbarMode, total, visible int, canShowAuto bool) bool {
	switch mode {
	case ScrollbarNever:
		return false
	case ScrollbarAlways:
		return true
	default:
		if !canShowAuto {
			return false
		}
		if total < 0 {
			total = 0
		}
		if visible < 0 {
			visible = 0
		}
		return total > visible
	}
}

func resolveScrollbarThumb(trackLen, visible, total, offset, minThumb int) (pos, length int) {
	if trackLen <= 0 || total <= 0 {
		return 0, 0
	}
	if visible < 0 {
		visible = 0
	}
	if visible > total {
		visible = total
	}
	if minThumb <= 0 {
		minThumb = 1
	}

	length = (trackLen * visible) / total
	if length < minThumb {
		length = minThumb
	}
	if length > trackLen {
		length = trackLen
	}

	if total <= visible || length >= trackLen {
		return 0, length
	}

	scrollRange := total - visible
	trackRange := trackLen - length
	offset = clampInt(offset, 0, scrollRange)
	// Rounded integer mapping keeps thumb endpoints deterministic.
	pos = (offset*trackRange + scrollRange/2) / scrollRange
	pos = clampInt(pos, 0, trackRange)
	return pos, length
}

func (m *Model) measureScrollExtents(lines []string, layout wrapLayoutCache, contentWidth int) (totalRows, totalCols int) {
	if contentWidth < 0 {
		contentWidth = 0
	}

	if m.layoutMatchesForExtentMeasurement(layout, len(lines), contentWidth) {
		totalRows = len(layout.rows)
		if m.cfg.WrapMode == WrapNone {
			for _, line := range layout.lines {
				if w := line.visual.VisualLen(); w > totalCols {
					totalCols = w
				}
			}
		}
		if totalRows <= 0 {
			totalRows = 1
		}
		return totalRows, totalCols
	}

	for row, rawLine := range lines {
		vt := m.virtualTextForRow(row, rawLine)
		vt = m.virtualTextWithGhost(row, rawLine, vt)
		visual := BuildVisualLine(rawLine, vt, m.cfg.TabWidth)
		if m.cfg.WrapMode == WrapNone {
			if w := visual.VisualLen(); w > totalCols {
				totalCols = w
			}
		}
		segments := wrapSegmentsForVisualLine(visual, m.cfg.WrapMode, contentWidth)
		if len(segments) == 0 {
			totalRows++
			continue
		}
		totalRows += len(segments)
	}

	if totalRows <= 0 {
		totalRows = 1
	}
	return totalRows, totalCols
}

func (m *Model) layoutMatchesForExtentMeasurement(layout wrapLayoutCache, lineCount, contentWidth int) bool {
	if !layout.valid {
		return false
	}
	if len(layout.lines) != lineCount {
		return false
	}
	if layout.key.wrapMode != m.cfg.WrapMode ||
		layout.key.tabWidth != m.cfg.TabWidth ||
		layout.key.focused != m.focused {
		return false
	}
	if m.cfg.WrapMode != WrapNone && layout.key.contentWidth != contentWidth {
		return false
	}
	if m.buf != nil && layout.key.textVersion != m.buf.TextVersion() {
		return false
	}
	return true
}
