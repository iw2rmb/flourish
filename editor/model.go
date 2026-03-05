package editor

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/iw2rmb/flourish/buffer"
	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

// Model is a Bubble Tea component that renders and interacts with a buffer.
type Model struct {
	cfg Config
	buf *buffer.Buffer

	focused bool

	completionState       CompletionState
	completionFilterClean bool     // set by recomputeCompletionQueryFromAnchor to skip redundant filter in syncFromBuffer
	completionLowerCache  []string // cached lowercased flattened text per completion item

	viewport viewport.Model
	// xOffset is the horizontal scroll offset in terminal cells. It is used only
	// when WrapMode==WrapNone.
	xOffset int

	lastBufVersion  uint64
	lastTextVersion uint64
	lastCursor      buffer.Pos
	lastSelection   buffer.Range
	lastSelectionOK bool

	ghostCache ghostCache

	mouseDragging bool
	mouseAnchor   buffer.Pos

	scrollbarDragAxis        scrollbarDragAxis
	scrollbarDragStartCell   int
	scrollbarDragStartOffset int

	layout wrapLayoutCache

	gutterInvalidationVersion uint64
	styleInvalidationVersion  uint64
	renderedRows              []string

	// Reusable per-render highlight bookkeeping, sized to logical line count.
	highlightVisible   []bool
	highlightsByLine   [][]HighlightSpan
	highlightsComputed []bool

	cachedLines    []string
	cachedLinesVer uint64
}

type scrollbarDragAxis int

const (
	dragNone scrollbarDragAxis = iota
	dragVertical
	dragHorizontal
)

func New(cfg Config) Model {
	if cfg.Style.isZero() {
		cfg.Style = DefaultStyle()
	}
	if cfg.KeyMap.isZero() {
		cfg.KeyMap = DefaultKeyMap()
	}
	cfg.CompletionKeyMap = normalizeCompletionKeyMap(cfg.CompletionKeyMap)
	cfg.CompletionInputMode = normalizeCompletionInputMode(cfg.CompletionInputMode)
	cfg.CompletionMaxVisibleRows = normalizeCompletionMaxVisibleRows(cfg.CompletionMaxVisibleRows)
	cfg.CompletionMaxWidth = normalizeCompletionMaxWidth(cfg.CompletionMaxWidth)
	if cfg.Scrollbar.MinThumb <= 0 {
		cfg.Scrollbar.MinThumb = 1
	}
	if cfg.TabWidth <= 0 {
		cfg.TabWidth = 4
	}

	m := Model{
		cfg:      cfg,
		buf:      buffer.New(cfg.Text, buffer.Options{HistoryLimit: cfg.HistoryLimit}),
		focused:  true,
		viewport: viewport.New(viewport.WithWidth(0), viewport.WithHeight(0)),
	}
	m.lastBufVersion = m.buf.Version()
	m.lastTextVersion = m.buf.TextVersion()
	m.lastCursor = m.buf.Cursor()
	m.lastSelection, m.lastSelectionOK = m.buf.Selection()
	m.rebuildContent()
	return m
}

func (m *Model) ghostForCursor() (Ghost, bool) {
	if m.buf == nil || m.cfg.GhostProvider == nil || !m.focused {
		return Ghost{}, false
	}

	cur := m.buf.Cursor()
	lines := m.ensureLines()
	if cur.Row < 0 || cur.Row >= len(lines) {
		return Ghost{}, false
	}

	lineText := lines[cur.Row]
	rawLen := graphemeutil.Count(lineText)
	col := clampInt(cur.GraphemeCol, 0, rawLen)
	return m.ghostFor(cur.Row, col, lineText, rawLen)
}

func (m Model) docVersion() uint64 {
	if m.buf == nil {
		return 0
	}
	return m.buf.Version()
}

func (m Model) Buffer() *buffer.Buffer { return m.buf }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) SetSize(width, height int) Model {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	if m.viewport.Width() == width && m.viewport.Height() == height {
		return m
	}
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height)

	m.rebuildContent()
	m.followCursorWithForce(true)
	return m
}

func (m Model) Focus() Model {
	if !m.focused {
		m.focused = true
		m.rebuildContent()
		m.followCursorWithForce(true)
	}
	return m
}

func (m Model) Blur() Model {
	if m.focused {
		m.focused = false
		m.rebuildContent()
	}
	return m
}

func (m Model) Focused() bool { return m.focused }

// InvalidateGutter marks gutter rendering as stale for all rows and rebuilds view
// content. Use this when gutter callbacks depend on host-managed state that changed
// outside of editor Update flow.
func (m Model) InvalidateGutter() Model {
	m.gutterInvalidationVersion++
	m.rebuildContent()
	return m
}

// InvalidateGutterRows marks specific gutter rows as stale and rebuilds view
// content for those rows. If partial row refresh cannot be applied, it falls back
// to a full content rebuild.
func (m Model) InvalidateGutterRows(rows ...int) Model {
	if len(rows) == 0 {
		return m
	}
	m.gutterInvalidationVersion++
	if !m.rebuildGutterRows(rows) {
		m.rebuildContent()
	}
	return m
}

// InvalidateStyles marks row/token style callback output as stale and rebuilds
// rendered content. Use this when style callbacks depend on host-managed state
// that changed outside editor Update flow.
func (m Model) InvalidateStyles() Model {
	m.styleInvalidationVersion++
	m.rebuildContent()
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.SetSize(msg.Width, msg.Height), nil
	case tea.MouseMsg:
		beforeVer := uint64(0)
		if m.buf != nil {
			beforeVer = m.buf.Version()
		}
		beforeYOffset := m.viewport.YOffset()

		var cmd tea.Cmd
		m, cmd = m.updateMouse(msg)
		// Rebuild content in case the host mutated the buffer outside of the editor.
		cursorChanged, versionChanged := m.syncFromBuffer()
		if cursorChanged || versionChanged {
			m.followCursorWithForce(false)
		}
		if m.cfg.Highlighter != nil && m.viewport.YOffset() != beforeYOffset {
			m.rebuildContent()
		}
		if m.cfg.OnChange != nil && m.buf != nil && m.buf.Version() != beforeVer {
			if ch, ok := m.buf.LastChange(); ok {
				m.cfg.OnChange(ch)
			}
		}
		// Don't force-follow cursor here; mouse scroll behavior is controlled by
		// ScrollPolicy.
		return m, cmd
	case tea.KeyPressMsg:
		beforeVer := uint64(0)
		if m.buf != nil {
			beforeVer = m.buf.Version()
		}

		m, cmd := m.updateKey(msg)
		cursorChanged, versionChanged := m.syncFromBuffer()
		if m.cfg.OnChange != nil && m.buf != nil && m.buf.Version() != beforeVer {
			if ch, ok := m.buf.LastChange(); ok {
				m.cfg.OnChange(ch)
			}
		}
		if cursorChanged || versionChanged {
			m.followCursorWithForce(false)
		}
		return m, cmd
	default:
		cursorChanged, versionChanged := m.syncFromBuffer()
		if cursorChanged || versionChanged {
			m.followCursorWithForce(true)
		}
		return m, nil
	}
}

func (m Model) View() tea.View {
	base := m.viewport.View()
	base = m.renderScrollbarChrome(base)
	if popup, ok := m.completionPopupRender(base); ok {
		return tea.NewView(popup.View)
	}
	return tea.NewView(base)
}

func (m *Model) syncFromBuffer() (cursorChanged bool, versionChanged bool) {
	if m.buf == nil {
		return false, false
	}

	prevCursor := m.lastCursor
	prevSelection := m.lastSelection
	prevSelectionOK := m.lastSelectionOK
	prevTextVersion := m.lastTextVersion

	ver := m.buf.Version()
	textVer := m.buf.TextVersion()
	cur := m.buf.Cursor()
	sel, selOK := m.buf.Selection()
	if ver == m.lastBufVersion &&
		textVer == m.lastTextVersion &&
		cur == m.lastCursor &&
		selOK == m.lastSelectionOK &&
		(!selOK || sel == m.lastSelection) {
		return false, false
	}

	cursorChanged = cur != prevCursor
	versionChanged = ver != m.lastBufVersion
	selectionChanged := selOK != prevSelectionOK || (selOK && sel != prevSelection)
	textChanged := textVer != prevTextVersion

	m.lastBufVersion = ver
	m.lastTextVersion = textVer
	m.lastCursor = cur
	m.lastSelection = sel
	m.lastSelectionOK = selOK

	if m.completionState.Visible && (cursorChanged || versionChanged) {
		if m.cursorOutsideCompletionAnchorToken() {
			m.completionState = CompletionState{}
		} else if !m.completionFilterClean {
			m.recomputeCompletionFilter(&m.completionState)
		}
	}
	m.completionFilterClean = false

	if textChanged {
		if !m.tryIncrementalTextRebuild(prevCursor, cur, prevSelection, prevSelectionOK, sel, selOK) {
			m.rebuildContent()
		}
		return cursorChanged, versionChanged
	}

	if cursorChanged || selectionChanged {
		if !m.rebuildCursorSelectionDirtyRows(
			prevCursor,
			cur,
			prevSelection,
			prevSelectionOK,
			sel,
			selOK,
		) {
			m.rebuildContent()
		}
		return cursorChanged, versionChanged
	}

	// Unknown non-text version mutation: preserve correctness with full rebuild.
	if versionChanged {
		m.rebuildContent()
	}
	return cursorChanged, versionChanged
}

func (m *Model) rebuildCursorSelectionDirtyRows(
	prevCursor buffer.Pos,
	nextCursor buffer.Pos,
	prevSel buffer.Range,
	prevSelOK bool,
	nextSel buffer.Range,
	nextSelOK bool,
) bool {
	if m.buf == nil {
		return false
	}

	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	if len(layout.rows) == 0 || len(m.renderedRows) != len(layout.rows) {
		return false
	}

	dirty := cursorSelectionDirtyRows(
		len(lines),
		prevCursor,
		nextCursor,
		prevSel,
		prevSelOK,
		nextSel,
		nextSelOK,
	)
	if len(dirty) == 0 {
		return true
	}

	if !m.refreshLayoutRows(lines, dirty) {
		return false
	}

	layout = m.layout
	if len(m.renderedRows) != len(layout.rows) {
		return false
	}

	metrics := m.resolveScrollbarMetrics(lines, layout)
	rendered := m.renderRows(lines, layout, metrics, dirty, true)
	if len(rendered) != len(layout.rows) {
		return false
	}
	m.setRenderedRows(rendered, metrics)
	return true
}

func cursorSelectionDirtyRows(
	lineCount int,
	prevCursor buffer.Pos,
	nextCursor buffer.Pos,
	prevSel buffer.Range,
	prevSelOK bool,
	nextSel buffer.Range,
	nextSelOK bool,
) map[int]struct{} {
	if lineCount <= 0 {
		return nil
	}

	dirty := make(map[int]struct{}, 4)
	addDirtyRow(dirty, lineCount, prevCursor.Row)
	addDirtyRow(dirty, lineCount, nextCursor.Row)
	addDirtyRangeRows(dirty, lineCount, prevSel, prevSelOK)
	addDirtyRangeRows(dirty, lineCount, nextSel, nextSelOK)
	return dirty
}

func addDirtyRow(dirty map[int]struct{}, lineCount, row int) {
	if row < 0 || row >= lineCount {
		return
	}
	dirty[row] = struct{}{}
}

func addDirtyRangeRows(dirty map[int]struct{}, lineCount int, r buffer.Range, ok bool) {
	if !ok || lineCount <= 0 {
		return
	}
	start := clampInt(r.Start.Row, 0, lineCount-1)
	end := clampInt(r.End.Row, 0, lineCount-1)
	if end < start {
		start, end = end, start
	}
	for row := start; row <= end; row++ {
		dirty[row] = struct{}{}
	}
}

func (m *Model) cursorOutsideCompletionAnchorToken() bool {
	if m.buf == nil || !m.completionState.Visible {
		return false
	}

	state := m.completionState
	cursor := m.buf.Cursor()
	if cursor.Row != state.Anchor.Row {
		return true
	}

	lines := m.ensureLines()
	if state.Anchor.Row < 0 || state.Anchor.Row >= len(lines) {
		return true
	}

	rowClusters := graphemeutil.Split(lines[state.Anchor.Row])
	startCol, endCol := completionAnchorTokenBounds(rowClusters, state.Anchor.GraphemeCol)
	return cursor.GraphemeCol < startCol || cursor.GraphemeCol > endCol
}

func (m *Model) rebuildContent() {
	m.invalidateLayoutCache()
	if m.buf == nil {
		m.setRenderedRows(nil, scrollbarMetrics{})
		return
	}
	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	metrics := m.resolveScrollbarMetrics(lines, layout)
	rows := m.renderRows(lines, layout, metrics, nil, false)
	m.setRenderedRows(rows, metrics)
}

// tryIncrementalTextRebuild attempts to rebuild only the lines that changed
// after a text mutation. Returns false if a full rebuild is needed.
func (m *Model) tryIncrementalTextRebuild(
	prevCursor buffer.Pos,
	nextCursor buffer.Pos,
	prevSel buffer.Range,
	prevSelOK bool,
	nextSel buffer.Range,
	nextSelOK bool,
) bool {
	if m.buf == nil || !m.layout.valid {
		return false
	}

	lines := m.ensureLines()
	if len(lines) != len(m.layout.lines) {
		return false
	}

	// Check that layout config hasn't changed.
	contentWidth := m.contentWidth(lines)
	k := m.layout.key
	if k.wrapMode != m.cfg.WrapMode ||
		k.tabWidth != m.cfg.TabWidth ||
		k.contentWidth != contentWidth ||
		k.focused != m.focused ||
		k.linkProvider != providerPtr(m.cfg.LinkProvider) ||
		k.linkSet != (m.cfg.LinkProvider != nil) {
		return false
	}

	// Collect text-dirty lines (rawLine changed).
	dirty := make(map[int]struct{}, 4)
	for i, rawLine := range lines {
		if rawLine != m.layout.lines[i].rawLine {
			dirty[i] = struct{}{}
		}
	}

	// Also dirty cursor rows for virtual text / ghost / link updates.
	addDirtyRow(dirty, len(lines), prevCursor.Row)
	addDirtyRow(dirty, len(lines), nextCursor.Row)
	addDirtyRangeRows(dirty, len(lines), prevSel, prevSelOK)
	addDirtyRangeRows(dirty, len(lines), nextSel, nextSelOK)

	if len(dirty) == 0 {
		m.layout.key.textVersion = m.buf.TextVersion()
		return true
	}

	if !m.refreshLayoutRows(lines, dirty) {
		return false
	}

	m.layout.key.textVersion = m.buf.TextVersion()

	if len(m.renderedRows) != len(m.layout.rows) {
		return false
	}

	metrics := m.resolveScrollbarMetrics(lines, m.layout)
	rendered := m.renderRows(lines, m.layout, metrics, dirty, true)
	if len(rendered) != len(m.layout.rows) {
		return false
	}
	m.setRenderedRows(rendered, metrics)
	return true
}

func (m *Model) setRenderedRows(rows []string, metrics scrollbarMetrics) {
	// Reuse the backing array when it has enough capacity.
	if cap(m.renderedRows) >= len(rows) {
		m.renderedRows = m.renderedRows[:len(rows)]
	} else {
		m.renderedRows = make([]string, len(rows))
	}
	copy(m.renderedRows, rows)

	content := strings.Join(rows, "\n")
	if metrics.showH {
		if content == "" {
			content = "\n"
		} else {
			content += "\n"
		}
	}
	m.viewport.SetContent(content)
}

func (m *Model) contentWidth(lines []string) int {
	return m.resolveScrollbarMetrics(lines, m.layout).contentWidth
}

func cursorCellForVisualLine(vl VisualLine, cursorCol int) int {
	cursorCol = clampInt(cursorCol, 0, vl.RawGraphemeLen)
	if cursorCol != vl.RawGraphemeLen {
		return vl.VisualCellForDocGraphemeCol(cursorCol)
	}

	// Cursor at EOL is rendered as a 1-cell placeholder inserted before any
	// virtual insertions anchored at the raw EOL.
	for _, tok := range vl.Tokens {
		if tok.Kind == VisualTokenVirtual && tok.DocStartGraphemeCol == vl.RawGraphemeLen {
			return tok.StartCell
		}
	}
	return vl.VisualLen()
}

func (m *Model) followCursorWithForce(force bool) {
	if m.buf == nil {
		return
	}
	cur := m.buf.Cursor()
	lines := m.ensureLines()
	layout := m.ensureLayoutCache(lines)
	metrics := m.resolveScrollbarMetrics(lines, layout)

	beforeYOffset := m.viewport.YOffset()
	newYOffset := metrics.yOffset
	if metrics.contentHeight > 0 {
		cursorVisualRow := cur.Row
		if vr, _, ok := layout.cursorVisualPosition(cur); ok {
			cursorVisualRow = vr
		}
		if cursorVisualRow < newYOffset {
			newYOffset = cursorVisualRow
		} else if cursorVisualRow >= newYOffset+metrics.contentHeight {
			newYOffset = cursorVisualRow - metrics.contentHeight + 1
		}
		maxYOffset := 0
		if metrics.totalRows > metrics.contentHeight {
			maxYOffset = metrics.totalRows - metrics.contentHeight
		}
		newYOffset = clampInt(newYOffset, 0, maxYOffset)
	}
	if newYOffset != beforeYOffset {
		m.viewport.SetYOffset(newYOffset)
		if m.cfg.Highlighter != nil {
			m.rebuildContent()
		}
	}

	if m.cfg.WrapMode != WrapNone {
		if m.xOffset != 0 {
			m.xOffset = 0
			m.rebuildContent()
		}
		return
	}

	if len(lines) == 0 || cur.Row < 0 || cur.Row >= len(lines) {
		if m.xOffset != 0 {
			m.xOffset = 0
			m.rebuildContent()
		}
		return
	}

	cw := metrics.contentWidth
	if cw <= 0 {
		if m.xOffset != 0 {
			m.xOffset = 0
			m.rebuildContent()
		}
		return
	}

	rawLine := lines[cur.Row]
	vt := m.virtualTextForRow(cur.Row, rawLine)
	vt = m.virtualTextWithGhost(cur.Row, rawLine, vt)
	vl := BuildVisualLine(rawLine, vt, m.cfg.TabWidth)

	cursorCell := cursorCellForVisualLine(vl, cur.GraphemeCol)
	newXOffset := metrics.xOffset
	if cursorCell < newXOffset {
		newXOffset = cursorCell
	} else if cursorCell >= newXOffset+cw {
		newXOffset = cursorCell - cw + 1
	}
	if newXOffset < 0 {
		newXOffset = 0
	}
	maxXOffset := 0
	if metrics.totalCols > cw {
		maxXOffset = metrics.totalCols - cw
	}
	newXOffset = clampInt(newXOffset, 0, maxXOffset)

	// When not forced, avoid shifting horizontally while the user is actively
	// mouse-dragging a selection (it makes hit-testing feel unstable).
	if !force && m.mouseDragging {
		return
	}

	if newXOffset != m.xOffset {
		m.xOffset = newXOffset
		m.rebuildContent()
	}
}

func (m *Model) virtualTextForRow(row int, rawLine string) VirtualText {
	if m.buf == nil || m.cfg.VirtualTextProvider == nil {
		return VirtualText{}
	}

	rawLen := graphemeutil.Count(rawLine)

	cursor := m.buf.Cursor()
	hasCursor := cursor.Row == row
	cursorCol := cursor.GraphemeCol
	if cursorCol < 0 {
		cursorCol = 0
	}
	if cursorCol > rawLen {
		cursorCol = rawLen
	}

	sel, selOK := m.buf.Selection()
	selStartCol, selEndCol, hasSel := selectionColsForRow(sel, selOK, row, rawLen)

	ctx := VirtualTextContext{
		Row:      row,
		LineText: rawLine,

		CursorGraphemeCol: cursorCol,
		HasCursor:         hasCursor,

		SelectionStartGraphemeCol: selStartCol,
		SelectionEndGraphemeCol:   selEndCol,
		HasSelection:              hasSel,

		DocID:      m.cfg.DocID,
		DocVersion: m.buf.Version(),
	}
	vt := m.cfg.VirtualTextProvider(ctx)
	return normalizeVirtualText(vt, rawLen)
}

func selectionColsForRow(sel buffer.Range, selOK bool, row int, lineLen int) (startCol, endCol int, hasSel bool) {
	if !selOK {
		return 0, 0, false
	}
	if row < sel.Start.Row || row > sel.End.Row {
		return 0, 0, false
	}

	startCol = 0
	endCol = lineLen
	if row == sel.Start.Row {
		startCol = sel.Start.GraphemeCol
	}
	if row == sel.End.Row {
		endCol = sel.End.GraphemeCol
	}
	if startCol < 0 {
		startCol = 0
	}
	if endCol < 0 {
		endCol = 0
	}
	if startCol > lineLen {
		startCol = lineLen
	}
	if endCol > lineLen {
		endCol = lineLen
	}
	if startCol > endCol {
		startCol, endCol = endCol, startCol
	}
	if startCol == endCol {
		return 0, 0, false
	}
	return startCol, endCol, true
}

func (m *Model) ensureLines() []string {
	if m.buf == nil {
		return nil
	}
	ver := m.buf.TextVersion()
	if m.cachedLines != nil && m.cachedLinesVer == ver {
		return m.cachedLines
	}
	m.cachedLines = m.buf.RawLines()
	m.cachedLinesVer = ver
	return m.cachedLines
}
