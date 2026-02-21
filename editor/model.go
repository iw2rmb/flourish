package editor

import (
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flourish/buffer"
	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

// Model is a Bubble Tea component that renders and interacts with a buffer.
//
// Phase 5 includes sizing, focus, viewport, optional gutter, and cursor rendering.
type Model struct {
	cfg Config
	buf *buffer.Buffer

	focused bool

	completionState CompletionState

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

	layout wrapLayoutCache

	gutterInvalidationVersion uint64
	renderedRows              []string
}

func New(cfg Config) Model {
	if reflect.DeepEqual(cfg.Style, Style{}) {
		cfg.Style = DefaultStyle()
	}
	if reflect.DeepEqual(cfg.KeyMap, KeyMap{}) {
		cfg.KeyMap = DefaultKeyMap()
	}
	cfg.CompletionKeyMap = normalizeCompletionKeyMap(cfg.CompletionKeyMap)
	cfg.CompletionInputMode = normalizeCompletionInputMode(cfg.CompletionInputMode)
	cfg.CompletionMaxVisibleRows = normalizeCompletionMaxVisibleRows(cfg.CompletionMaxVisibleRows)
	cfg.CompletionMaxWidth = normalizeCompletionMaxWidth(cfg.CompletionMaxWidth)
	if cfg.TabWidth <= 0 {
		cfg.TabWidth = 4
	}

	m := Model{
		cfg:      cfg,
		buf:      buffer.New(cfg.Text, buffer.Options{HistoryLimit: cfg.HistoryLimit}),
		focused:  true,
		viewport: viewport.New(0, 0),
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
	lines := rawLinesFromBufferText(m.buf.Text())
	if cur.Row < 0 || cur.Row >= len(lines) {
		return Ghost{}, false
	}

	lineText := lines[cur.Row]
	rawLen := graphemeutil.Count(lineText)
	col := clampInt(cur.GraphemeCol, 0, rawLen)
	return m.ghostFor(cur.Row, col, lineText, rawLen)
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
	m.viewport.Width = width
	m.viewport.Height = height

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

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.SetSize(msg.Width, msg.Height), nil
	case tea.MouseMsg:
		beforeVer := uint64(0)
		if m.buf != nil {
			beforeVer = m.buf.Version()
		}
		beforeYOffset := m.viewport.YOffset

		var cmd tea.Cmd
		m, cmd = m.updateMouse(msg)
		// Rebuild content in case the host mutated the buffer outside of the editor.
		cursorChanged, versionChanged := m.syncFromBuffer()
		if cursorChanged || versionChanged {
			m.followCursorWithForce(false)
		}
		if m.cfg.Highlighter != nil && m.viewport.YOffset != beforeYOffset {
			m.rebuildContent()
		}
		if m.cfg.OnChange != nil && m.buf != nil && m.buf.Version() != beforeVer {
			if ev, ok := buildChangeEvent(m.buf); ok {
				m.cfg.OnChange(ev)
			}
		}
		// Don't force-follow cursor here; mouse scroll behavior is controlled by
		// ScrollPolicy.
		return m, cmd
	case tea.KeyMsg:
		beforeVer := uint64(0)
		if m.buf != nil {
			beforeVer = m.buf.Version()
		}

		m, cmd := m.updateKey(msg)
		cursorChanged, versionChanged := m.syncFromBuffer()
		if m.cfg.OnChange != nil && m.buf != nil && m.buf.Version() != beforeVer {
			if ev, ok := buildChangeEvent(m.buf); ok {
				m.cfg.OnChange(ev)
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

func (m Model) View() string {
	base := m.viewport.View()
	if popup, ok := m.completionPopupRender(base); ok {
		return popup.View
	}
	return base
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
		} else {
			m.recomputeCompletionFilter(&m.completionState)
		}
	}

	if textChanged {
		m.rebuildContent()
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

	lines := rawLinesFromBufferText(m.buf.Text())
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

	rendered := m.renderRows(lines, layout, dirty, true)
	if len(rendered) != len(layout.rows) {
		return false
	}
	m.setRenderedRows(rendered)
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

	lines := strings.Split(m.buf.Text(), "\n")
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
		m.setRenderedRows(nil)
		return
	}
	lines := rawLinesFromBufferText(m.buf.Text())
	layout := m.ensureLayoutCache(lines)
	rows := m.renderRows(lines, layout, nil, false)
	m.setRenderedRows(rows)
}

func (m *Model) setRenderedRows(rows []string) {
	m.renderedRows = append([]string(nil), rows...)
	m.viewport.SetContent(strings.Join(rows, "\n"))
}

func (m Model) contentWidth(lineCount int) int {
	w := m.viewport.Width - m.viewport.Style.GetHorizontalFrameSize() - m.resolvedGutterWidth(lineCount)
	if w < 0 {
		w = 0
	}
	return w
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
	lines := rawLinesFromBufferText(m.buf.Text())
	layout := m.ensureLayoutCache(lines)

	h := m.viewport.Height - m.viewport.Style.GetVerticalFrameSize()
	beforeYOffset := m.viewport.YOffset
	newYOffset := beforeYOffset
	if h > 0 {
		cursorVisualRow := cur.Row
		if vr, _, ok := layout.cursorVisualPosition(cur); ok {
			cursorVisualRow = vr
		}
		if cursorVisualRow < beforeYOffset {
			newYOffset = cursorVisualRow
		} else if cursorVisualRow >= beforeYOffset+h {
			newYOffset = cursorVisualRow - h + 1
		}
		if newYOffset != beforeYOffset {
			m.viewport.SetYOffset(newYOffset)
			if m.cfg.Highlighter != nil {
				m.rebuildContent()
			}
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

	cw := m.contentWidth(len(lines))
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
	newXOffset := m.xOffset
	if cursorCell < newXOffset {
		newXOffset = cursorCell
	} else if cursorCell >= newXOffset+cw {
		newXOffset = cursorCell - cw + 1
	}
	if newXOffset < 0 {
		newXOffset = 0
	}

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

func rawLinesFromBufferText(text string) []string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
