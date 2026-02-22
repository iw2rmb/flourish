package editor

import (
	"strings"

	"github.com/iw2rmb/flourish/buffer"
)

func (m *Model) recomputeCompletionFilter(state *CompletionState) {
	if state == nil {
		return
	}

	// Rebuild lowercased text cache when item list changes.
	if len(m.completionLowerCache) != len(state.Items) {
		m.completionLowerCache = make([]string, len(state.Items))
		for i := range state.Items {
			m.completionLowerCache[i] = strings.ToLower(flattenCompletionItemText(state.Items[i]))
		}
	}

	ctx := CompletionFilterContext{
		Query:      state.Query,
		Items:      state.Items,
		Cursor:     m.completionFilterCursor(),
		DocID:      m.cfg.DocID,
		DocVersion: m.docVersion(),
		lowerCache: m.completionLowerCache,
	}

	result := CompletionFilterResult{}
	if m.cfg.CompletionFilter != nil {
		result = m.cfg.CompletionFilter(ctx)
	} else {
		result = defaultCompletionFilter(ctx)
	}

	state.VisibleIndices = sanitizeCompletionVisibleIndices(result.VisibleIndices, len(state.Items))
	state.Selected = clampCompletionSelected(result.SelectedIndex, len(state.VisibleIndices))
}

func (m *Model) completionFilterCursor() buffer.Pos {
	if m == nil || m.buf == nil {
		return buffer.Pos{}
	}
	return m.buf.Cursor()
}

func defaultCompletionFilter(ctx CompletionFilterContext) CompletionFilterResult {
	query := strings.ToLower(ctx.Query)
	visible := make([]int, 0, len(ctx.Items))
	lowerCache := ctx.lowerCache
	for i := range ctx.Items {
		var itemText string
		if i < len(lowerCache) {
			itemText = lowerCache[i]
		} else {
			itemText = strings.ToLower(flattenCompletionItemText(ctx.Items[i]))
		}
		if strings.Contains(itemText, query) {
			visible = append(visible, i)
		}
	}
	return CompletionFilterResult{
		VisibleIndices: visible,
		SelectedIndex:  0,
	}
}

func flattenCompletionItemText(item CompletionItem) string {
	var sb strings.Builder
	appendSegments := func(segments []CompletionSegment) {
		for _, seg := range segments {
			sb.WriteString(seg.Text)
		}
	}
	appendSegments(item.Prefix)
	appendSegments(item.Label)
	appendSegments(item.Detail)
	return sb.String()
}
