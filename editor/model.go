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
// Phase 5 includes sizing, focus, viewport, line numbers, and cursor rendering.
type Model struct {
	cfg Config
	buf *buffer.Buffer

	focused bool

	viewport viewport.Model
	// xOffset is the horizontal scroll offset in terminal cells. It is used only
	// when WrapMode==WrapNone.
	xOffset int

	lastBufVersion uint64
	lastCursor     buffer.Pos

	ghostCache ghostCache

	mouseDragging bool
	mouseAnchor   buffer.Pos

	layout wrapLayoutCache
}

func New(cfg Config) Model {
	if reflect.DeepEqual(cfg.Style, Style{}) {
		cfg.Style = DefaultStyle()
	}
	if reflect.DeepEqual(cfg.KeyMap, KeyMap{}) {
		cfg.KeyMap = DefaultKeyMap()
	}
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
	m.lastCursor = m.buf.Cursor()
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

func (m Model) View() string { return m.viewport.View() }

func (m *Model) syncFromBuffer() (cursorChanged bool, versionChanged bool) {
	if m.buf == nil {
		return false, false
	}
	ver := m.buf.Version()
	cur := m.buf.Cursor()
	if ver == m.lastBufVersion && cur == m.lastCursor {
		return false, false
	}
	cursorChanged = cur != m.lastCursor
	versionChanged = ver != m.lastBufVersion
	m.lastBufVersion = ver
	m.lastCursor = cur
	m.rebuildContent()
	return cursorChanged, versionChanged
}

func (m *Model) rebuildContent() {
	m.invalidateLayoutCache()
	m.viewport.SetContent(m.renderContent())
}

func (m Model) contentWidth(lineCount int) int {
	w := m.viewport.Width - m.viewport.Style.GetHorizontalFrameSize() - m.gutterWidth(lineCount)
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
