package editor

import (
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iw2rmb/flouris/buffer"
)

// Model is a Bubble Tea component that renders and interacts with a buffer.
//
// Phase 5 includes sizing, focus, viewport, line numbers, and cursor rendering.
type Model struct {
	cfg Config
	buf *buffer.Buffer

	focused bool

	viewport viewport.Model

	lastBufVersion uint64
	lastCursor     buffer.Pos

	ghostCache ghostCache

	mouseDragging bool
	mouseAnchor   buffer.Pos
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
	rawLen := len([]rune(lineText))
	col := clampInt(cur.Col, 0, rawLen)
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
		m.syncFromBuffer()
		if m.cfg.Highlighter != nil && m.viewport.YOffset != beforeYOffset {
			m.rebuildContent()
		}
		if m.cfg.OnChange != nil && m.buf != nil && m.buf.Version() != beforeVer {
			m.cfg.OnChange(buildChangeEvent(m.buf))
		}
		// Don't force-follow cursor here; allow manual scrolling via mouse wheel.
		return m, cmd
	case tea.KeyMsg:
		beforeVer := uint64(0)
		if m.buf != nil {
			beforeVer = m.buf.Version()
		}

		m, cmd := m.updateKey(msg)
		cursorChanged := m.syncFromBuffer()
		if m.cfg.OnChange != nil && m.buf != nil && m.buf.Version() != beforeVer {
			m.cfg.OnChange(buildChangeEvent(m.buf))
		}
		if cursorChanged {
			m.followCursorWithForce(false)
		}
		return m, cmd
	default:
		cursorChanged := m.syncFromBuffer()
		if cursorChanged {
			m.followCursorWithForce(true)
		}
		return m, nil
	}
}

func (m Model) View() string { return m.viewport.View() }

func (m *Model) syncFromBuffer() (cursorChanged bool) {
	if m.buf == nil {
		return false
	}
	ver := m.buf.Version()
	cur := m.buf.Cursor()
	if ver == m.lastBufVersion && cur == m.lastCursor {
		return false
	}
	cursorChanged = cur != m.lastCursor
	m.lastBufVersion = ver
	m.lastCursor = cur
	m.rebuildContent()
	return cursorChanged
}

func (m *Model) rebuildContent() {
	m.viewport.SetContent(m.renderContent())
}

func (m *Model) followCursorWithForce(force bool) {
	if m.buf == nil {
		return
	}
	cur := m.buf.Cursor()
	h := m.viewport.Height - m.viewport.Style.GetVerticalFrameSize()
	if h <= 0 {
		return
	}

	beforeYOffset := m.viewport.YOffset
	newYOffset := beforeYOffset
	if cur.Row < beforeYOffset {
		newYOffset = cur.Row
	} else if cur.Row >= beforeYOffset+h {
		newYOffset = cur.Row - h + 1
	}
	if newYOffset != beforeYOffset {
		m.viewport.SetYOffset(newYOffset)
		if m.cfg.Highlighter != nil {
			m.rebuildContent()
		}
	}
}

func (m *Model) virtualTextForRow(row int, rawLine string) VirtualText {
	if m.buf == nil || m.cfg.VirtualTextProvider == nil {
		return VirtualText{}
	}

	rawRunes := []rune(rawLine)
	rawLen := len(rawRunes)

	cursor := m.buf.Cursor()
	hasCursor := cursor.Row == row
	cursorCol := cursor.Col
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

		CursorCol: cursorCol,
		HasCursor: hasCursor,

		SelectionStartCol: selStartCol,
		SelectionEndCol:   selEndCol,
		HasSelection:      hasSel,

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
		startCol = sel.Start.Col
	}
	if row == sel.End.Row {
		endCol = sel.End.Col
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
