package editor

import (
	"strings"

	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type completionPopupRender struct {
	View string
}

func (m Model) completionPopupRender(base string) (completionPopupRender, bool) {
	state := m.completionState // read-only; no clone needed for rendering
	if !state.Visible || m.buf == nil {
		return completionPopupRender{}, false
	}

	viewportWidth := m.viewport.Width - m.viewport.Style.GetHorizontalFrameSize()
	viewportHeight := m.visibleRowCount()
	if viewportWidth <= 0 || viewportHeight <= 0 {
		return completionPopupRender{}, false
	}

	anchorX, anchorY, ok := m.DocToScreen(state.Anchor)
	if !ok {
		return completionPopupRender{}, false
	}

	visible := sanitizeCompletionVisibleIndices(state.VisibleIndices, len(state.Items))
	if len(visible) == 0 {
		return completionPopupRender{}, false
	}

	maxRows := normalizeCompletionMaxVisibleRows(m.cfg.CompletionMaxVisibleRows)
	if maxRows <= 0 {
		return completionPopupRender{}, false
	}
	targetRows := min(maxRows, len(visible))

	belowAvail := max(viewportHeight-(anchorY+1), 0)
	aboveAvail := max(anchorY, 0)
	showBelow := true
	rowCount := targetRows
	if rowCount > belowAvail {
		if aboveAvail >= rowCount {
			showBelow = false
		} else if aboveAvail > belowAvail {
			showBelow = false
			rowCount = aboveAvail
		} else {
			rowCount = belowAvail
		}
	}
	if rowCount <= 0 {
		return completionPopupRender{}, false
	}

	itemIndices := visible
	selected := clampCompletionSelected(state.Selected, len(visible))
	start := 0
	if len(itemIndices) > rowCount {
		start = clampInt(selected-rowCount+1, 0, len(itemIndices)-rowCount)
		itemIndices = itemIndices[start : start+rowCount]
	}

	widthCap := min(normalizeCompletionMaxWidth(m.cfg.CompletionMaxWidth), viewportWidth)
	if widthCap <= 0 {
		return completionPopupRender{}, false
	}

	// Single pass: compute segments for each item, measure width, and store for rendering.
	type itemEntry struct {
		item     CompletionItem
		segments []CompletionSegment
	}
	entries := make([]itemEntry, len(itemIndices))
	popupWidth := 0
	for i, idx := range itemIndices {
		item := state.Items[idx]
		segs := completionItemSegments(item)
		entries[i] = itemEntry{item: item, segments: segs}
		if w := completionSegmentsCellWidth(segs); w > popupWidth {
			popupWidth = w
		}
	}
	if popupWidth <= 0 {
		return completionPopupRender{}, false
	}
	if popupWidth > widthCap {
		popupWidth = widthCap
	}

	rendered := make([]string, 0, len(itemIndices))
	for row, e := range entries {
		selectedRow := row == selected-start
		rendered = append(rendered, m.renderCompletionPopupRowFromSegments(e.item, e.segments, selectedRow, popupWidth))
	}

	y := anchorY + 1
	if !showBelow {
		y = anchorY - len(rendered)
	}
	if y < 0 {
		y = 0
	}
	maxY := viewportHeight - len(rendered)
	if maxY < 0 {
		maxY = 0
	}
	if y > maxY {
		y = maxY
	}

	x := anchorX
	maxX := viewportWidth - popupWidth
	if maxX < 0 {
		maxX = 0
	}
	x = clampInt(x, 0, maxX)

	leftFrame := m.viewport.Style.GetMarginLeft() + m.viewport.Style.GetBorderLeftSize() + m.viewport.Style.GetPaddingLeft()
	topFrame := m.viewport.Style.GetMarginTop() + m.viewport.Style.GetBorderTopSize() + m.viewport.Style.GetPaddingTop()

	return completionPopupRender{
		View: overlay.Composite(
			strings.Join(rendered, "\n"),
			base,
			overlay.Left,
			overlay.Top,
			leftFrame+x,
			topFrame+y,
		),
	}, true
}

func (m Model) renderCompletionPopupRowFromSegments(item CompletionItem, precomputed []CompletionSegment, selected bool, width int) string {
	segments := truncateCompletionSegments(precomputed, width)
	base := completionRowBaseStyle(m.cfg.Style, selected)

	var sb strings.Builder
	used := 0
	for _, seg := range segments {
		text := sanitizeSegmentText(seg.Text)
		if text == "" {
			continue
		}
		segStyle := resolveCompletionSegmentStyle(base, m.cfg.CompletionStyleForKey, item, seg)
		sb.WriteString(segStyle.Render(text))
		if seg.cellWidth >= 0 {
			used += seg.cellWidth
		} else {
			used += computeSegmentCellWidth(text, used)
		}
	}

	if used < width {
		sb.WriteString(base.Render(spaceString(width - used)))
	}

	return sb.String()
}

func completionItemSegments(item CompletionItem) []CompletionSegment {
	prefix := item.Prefix
	label := item.Label
	detail := item.Detail

	out := make([]CompletionSegment, 0, len(prefix)+len(label)+len(detail)+2)
	appendGroup := func(group []CompletionSegment) {
		for _, seg := range group {
			text := sanitizeSegmentText(seg.Text)
			if text == "" {
				continue
			}
			out = append(out, CompletionSegment{Text: text, StyleKey: seg.StyleKey, cellWidth: -1})
		}
	}

	appendGroup(prefix)
	if len(out) > 0 && len(label) > 0 {
		out = append(out, CompletionSegment{Text: " ", cellWidth: -1})
	}
	appendGroup(label)
	if len(out) > 0 && len(detail) > 0 {
		out = append(out, CompletionSegment{Text: " ", cellWidth: -1})
	}
	appendGroup(detail)

	// Precompute cell widths so rendering doesn't have to re-split graphemes.
	pos := 0
	for i := range out {
		w := computeSegmentCellWidth(out[i].Text, pos)
		out[i].cellWidth = w
		pos += w
	}
	return out
}

func completionSegmentsCellWidth(segments []CompletionSegment) int {
	width := 0
	for _, seg := range segments {
		if seg.cellWidth >= 0 {
			width += seg.cellWidth
		} else {
			width += computeSegmentCellWidth(seg.Text, width)
		}
	}
	return width
}

func computeSegmentCellWidth(text string, start int) int {
	used := start
	begin := used
	for _, gr := range splitGraphemeBoundaries(sanitizeSegmentText(text)) {
		w := graphemeCellWidth(gr.Text, used, 4)
		if w < 1 {
			w = 1
		}
		used += w
	}
	return used - begin
}
