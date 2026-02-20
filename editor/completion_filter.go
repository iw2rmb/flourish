package editor

import (
	"strings"

	"github.com/iw2rmb/flourish/buffer"
)

func (m *Model) recomputeCompletionFilter(state *CompletionState) {
	if state == nil {
		return
	}

	ctx := CompletionFilterContext{
		Query:      state.Query,
		Items:      cloneCompletionItems(state.Items),
		Cursor:     m.completionFilterCursor(),
		DocID:      m.cfg.DocID,
		DocVersion: m.docVersion(),
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
	for i := range ctx.Items {
		itemText := strings.ToLower(flattenCompletionItemText(ctx.Items[i]))
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
