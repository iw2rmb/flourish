package editor

import (
	"strings"

	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type completionPopupRender struct {
	View string
}

func (m Model) completionPopupRender(base string) (completionPopupRender, bool) {
	state := cloneCompletionState(m.completionState)
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
	targetRows := minInt(maxRows, len(visible))

	belowAvail := maxInt(viewportHeight-(anchorY+1), 0)
	aboveAvail := maxInt(anchorY, 0)
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
	if len(itemIndices) > rowCount {
		itemIndices = itemIndices[:rowCount]
	}

	widthCap := minInt(normalizeCompletionMaxWidth(m.cfg.CompletionMaxWidth), viewportWidth)
	if widthCap <= 0 {
		return completionPopupRender{}, false
	}

	popupWidth := 0
	for _, idx := range itemIndices {
		item := state.Items[idx]
		if w := completionSegmentsCellWidth(completionItemSegments(item)); w > popupWidth {
			popupWidth = w
		}
	}
	if popupWidth <= 0 {
		return completionPopupRender{}, false
	}
	if popupWidth > widthCap {
		popupWidth = widthCap
	}

	selected := clampCompletionSelected(state.Selected, len(visible))
	rendered := make([]string, 0, len(itemIndices))
	for row, idx := range itemIndices {
		item := state.Items[idx]
		selectedRow := row == selected
		rendered = append(rendered, m.renderCompletionPopupRow(item, selectedRow, popupWidth))
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

func (m Model) renderCompletionPopupRow(item CompletionItem, selected bool, width int) string {
	segments := truncateCompletionSegments(completionItemSegments(item), width)
	base := completionRowBaseStyle(m.cfg.Style, selected)

	var sb strings.Builder
	used := 0
	for _, seg := range segments {
		text := sanitizeCompletionSegmentText(seg.Text)
		if text == "" {
			continue
		}
		segStyle := resolveCompletionSegmentStyle(base, m.cfg.CompletionStyleForKey, item, seg)
		sb.WriteString(segStyle.Render(text))
		used += completionSegmentCellWidth(text, used)
	}

	if used < width {
		sb.WriteString(base.Render(strings.Repeat(" ", width-used)))
	}

	return sb.String()
}

func completionItemSegments(item CompletionItem) []CompletionSegment {
	prefix := cloneCompletionSegments(item.Prefix)
	label := cloneCompletionSegments(item.Label)
	detail := cloneCompletionSegments(item.Detail)

	out := make([]CompletionSegment, 0, len(prefix)+len(label)+len(detail)+2)
	appendGroup := func(group []CompletionSegment) {
		for _, seg := range group {
			text := sanitizeCompletionSegmentText(seg.Text)
			if text == "" {
				continue
			}
			out = append(out, CompletionSegment{Text: text, StyleKey: seg.StyleKey})
		}
	}

	appendGroup(prefix)
	if len(out) > 0 && len(label) > 0 {
		out = append(out, CompletionSegment{Text: " "})
	}
	appendGroup(label)
	if len(out) > 0 && len(detail) > 0 {
		out = append(out, CompletionSegment{Text: " "})
	}
	appendGroup(detail)
	return out
}

func completionSegmentsCellWidth(segments []CompletionSegment) int {
	width := 0
	for _, seg := range segments {
		width += completionSegmentCellWidth(seg.Text, width)
	}
	return width
}

func completionSegmentCellWidth(text string, start int) int {
	used := start
	begin := used
	for _, gr := range splitGraphemeBoundaries(sanitizeCompletionSegmentText(text)) {
		w := graphemeCellWidth(gr.Text, used, 4)
		if w < 1 {
			w = 1
		}
		used += w
	}
	return used - begin
}
