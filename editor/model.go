package editor

import (
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
}

func New(cfg Config) Model {
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
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		// Rebuild content in case the host mutated the buffer outside of the editor.
		m.syncFromBuffer()
		// Don't force-follow cursor here; allow manual scrolling via mouse wheel.
		return m, cmd
	default:
		// No internal key handling in phase 5. Hosts may drive edits by mutating the buffer.
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

	y := m.viewport.YOffset
	if cur.Row < y {
		m.viewport.SetYOffset(cur.Row)
		return
	}
	if cur.Row >= y+h {
		m.viewport.SetYOffset(cur.Row - h + 1)
		return
	}
}
