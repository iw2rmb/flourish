package editor

import (
	"encoding/binary"
	"hash/fnv"
	"reflect"

	"github.com/iw2rmb/flourish/buffer"
)

type SnapshotToken uint64

type RowMap struct {
	ScreenRow        int
	DocRow           int
	SegmentIndex     int
	DocStartGrapheme int
	DocEndGrapheme   int
	VisibleDocCols   []int
}

type RenderSnapshot struct {
	Token         SnapshotToken
	BufferVersion uint64
	Viewport      ViewportState
	Rows          []RowMap
}

type snapshotSignature struct {
	bufVersion uint64
	cursor     buffer.Pos
	sel        buffer.Range
	selOK      bool

	viewportWidth             int
	viewportHeight            int
	viewportStyleH            int
	viewportStyleV            int
	viewportYOffset           int
	xOffset                   int
	wrapMode                  WrapMode
	tabWidth                  int
	focused                   bool
	docID                     string
	gutterInvalidationVersion uint64

	gutterWidthProvider uintptr
	gutterCellProvider  uintptr
	gutterWidthSet      bool
	gutterCellSet       bool

	virtualProvider uintptr
	ghostProvider   uintptr
	highlighter     uintptr
	virtualSet      bool
	ghostSet        bool
	highlighterSet  bool
}

func providerPtr(v any) uintptr {
	if v == nil {
		return 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Func, reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Slice, reflect.Chan:
		return rv.Pointer()
	default:
		return 0
	}
}

func (m *Model) currentSnapshotSignature() snapshotSignature {
	sig := snapshotSignature{
		viewportWidth:             m.viewport.Width,
		viewportHeight:            m.viewport.Height,
		viewportStyleH:            m.viewport.Style.GetHorizontalFrameSize(),
		viewportStyleV:            m.viewport.Style.GetVerticalFrameSize(),
		viewportYOffset:           m.viewport.YOffset,
		xOffset:                   m.xOffset,
		wrapMode:                  m.cfg.WrapMode,
		tabWidth:                  m.cfg.TabWidth,
		focused:                   m.focused,
		docID:                     m.cfg.DocID,
		gutterInvalidationVersion: m.gutterInvalidationVersion,
		gutterWidthProvider:       providerPtr(m.cfg.Gutter.Width),
		gutterCellProvider:        providerPtr(m.cfg.Gutter.Cell),
		gutterWidthSet:            m.cfg.Gutter.Width != nil,
		gutterCellSet:             m.cfg.Gutter.Cell != nil,
		virtualProvider:           providerPtr(m.cfg.VirtualTextProvider),
		ghostProvider:             providerPtr(m.cfg.GhostProvider),
		highlighter:               providerPtr(m.cfg.Highlighter),
		virtualSet:                m.cfg.VirtualTextProvider != nil,
		ghostSet:                  m.cfg.GhostProvider != nil,
		highlighterSet:            m.cfg.Highlighter != nil,
	}

	if m.buf != nil {
		sig.bufVersion = m.buf.Version()
		sig.cursor = m.buf.Cursor()
		sig.sel, sig.selOK = m.buf.Selection()
	}
	return sig
}

func hashSnapshotSignature(sig snapshotSignature) SnapshotToken {
	h := fnv.New64a()
	writeU64 := func(v uint64) {
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], v)
		_, _ = h.Write(b[:])
	}
	writeI := func(v int) { writeU64(uint64(v)) }
	writeB := func(v bool) {
		if v {
			writeU64(1)
			return
		}
		writeU64(0)
	}
	writeS := func(v string) {
		writeU64(uint64(len(v)))
		_, _ = h.Write([]byte(v))
	}

	writeU64(sig.bufVersion)
	writeI(sig.cursor.Row)
	writeI(sig.cursor.GraphemeCol)
	writeI(sig.sel.Start.Row)
	writeI(sig.sel.Start.GraphemeCol)
	writeI(sig.sel.End.Row)
	writeI(sig.sel.End.GraphemeCol)
	writeB(sig.selOK)
	writeI(sig.viewportWidth)
	writeI(sig.viewportHeight)
	writeI(sig.viewportStyleH)
	writeI(sig.viewportStyleV)
	writeI(sig.viewportYOffset)
	writeI(sig.xOffset)
	writeI(int(sig.wrapMode))
	writeI(sig.tabWidth)
	writeB(sig.focused)
	writeS(sig.docID)
	writeU64(sig.gutterInvalidationVersion)
	writeU64(uint64(sig.gutterWidthProvider))
	writeU64(uint64(sig.gutterCellProvider))
	writeB(sig.gutterWidthSet)
	writeB(sig.gutterCellSet)
	writeU64(uint64(sig.virtualProvider))
	writeU64(uint64(sig.ghostProvider))
	writeU64(uint64(sig.highlighter))
	writeB(sig.virtualSet)
	writeB(sig.ghostSet)
	writeB(sig.highlighterSet)

	tok := SnapshotToken(h.Sum64())
	if tok == 0 {
		return 1
	}
	return tok
}

func (m *Model) buildRenderSnapshot(token SnapshotToken) RenderSnapshot {
	m.syncFromBuffer()

	s := RenderSnapshot{
		Token:    token,
		Viewport: m.ViewportState(),
	}
	if m.buf == nil {
		return s
	}
	s.BufferVersion = m.buf.Version()

	lines := rawLinesFromBufferText(m.buf.Text())
	layout := m.ensureLayoutCache(lines)
	if len(layout.rows) == 0 {
		return s
	}

	start := clampInt(s.Viewport.TopVisualRow, 0, len(layout.rows)-1)
	end := start + s.Viewport.VisibleRows
	if end > len(layout.rows) {
		end = len(layout.rows)
	}
	if end < start {
		end = start
	}

	s.Rows = make([]RowMap, 0, end-start)
	for visualRow := start; visualRow < end; visualRow++ {
		docRow, line, seg, segIdx, ok := layout.lineAndSegmentAt(visualRow)
		if !ok {
			continue
		}
		row := RowMap{
			ScreenRow:        visualRow - s.Viewport.TopVisualRow,
			DocRow:           docRow,
			SegmentIndex:     segIdx,
			DocStartGrapheme: seg.StartGraphemeCol,
			DocEndGrapheme:   seg.EndGraphemeCol,
		}
		if seg.endCell > seg.startCell {
			row.VisibleDocCols = make([]int, 0, seg.endCell-seg.startCell)
			for cell := seg.startCell; cell < seg.endCell; cell++ {
				row.VisibleDocCols = append(row.VisibleDocCols, line.visual.DocGraphemeColForVisualCell(cell))
			}
		}
		s.Rows = append(s.Rows, row)
	}

	return s
}

func cloneRenderSnapshot(in RenderSnapshot) RenderSnapshot {
	out := in
	if len(in.Rows) == 0 {
		out.Rows = nil
		return out
	}
	out.Rows = make([]RowMap, len(in.Rows))
	for i := range in.Rows {
		out.Rows[i] = in.Rows[i]
		if len(in.Rows[i].VisibleDocCols) > 0 {
			out.Rows[i].VisibleDocCols = append([]int(nil), in.Rows[i].VisibleDocCols...)
		}
	}
	return out
}

func (m Model) RenderSnapshot() RenderSnapshot {
	sig := (&m).currentSnapshotSignature()
	tok := hashSnapshotSignature(sig)
	return cloneRenderSnapshot((&m).buildRenderSnapshot(tok))
}

func (m Model) snapshotMatchesCurrent(s RenderSnapshot) bool {
	if s.Token == 0 {
		return false
	}
	sig := (&m).currentSnapshotSignature()
	return s.Token == hashSnapshotSignature(sig)
}

func (m Model) ScreenToDocWithSnapshot(s RenderSnapshot, x, y int) (buffer.Pos, bool) {
	if m.buf == nil || len(s.Rows) == 0 {
		return buffer.Pos{}, false
	}
	if !m.snapshotMatchesCurrent(s) {
		return buffer.Pos{}, false
	}
	return (&m).screenToDocPos(x, y), true
}

func (m Model) DocToScreenWithSnapshot(s RenderSnapshot, pos buffer.Pos) (x int, y int, ok bool) {
	if m.buf == nil || len(s.Rows) == 0 {
		return 0, 0, false
	}
	if !m.snapshotMatchesCurrent(s) {
		return 0, 0, false
	}
	return (&m).docToScreenPos(pos)
}
