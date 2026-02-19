package editor

import (
	"github.com/iw2rmb/flourish/buffer"
	graphemeutil "github.com/iw2rmb/flourish/internal/grapheme"
)

type GhostContext struct {
	Row         int
	GraphemeCol int // grapheme index in the line
	LineText    string
	IsEndOfLine bool

	// Optional host metadata for caching.
	DocID      string
	DocVersion uint64
}

type Ghost struct {
	Text     string
	StyleKey string
	Edits    []buffer.TextEdit // deterministic apply
}

type GhostProvider func(ctx GhostContext) (Ghost, bool)

type GhostAccept struct {
	AcceptTab   bool
	AcceptRight bool
}

func normalizeGhostAccept(a GhostAccept) GhostAccept {
	// v0 default per design/api.md: Tab and Right.
	if !a.AcceptTab && !a.AcceptRight {
		return GhostAccept{AcceptTab: true, AcceptRight: true}
	}
	return a
}

type ghostCacheKey struct {
	docID      string
	docVersion uint64
	row        int
	col        int
}

type ghostCache struct {
	valid   bool
	key     ghostCacheKey
	present bool
	ghost   Ghost
}

func (c *ghostCache) get(key ghostCacheKey) (Ghost, bool) {
	if !c.valid || c.key != key {
		return Ghost{}, false
	}
	return c.ghost, c.present
}

func (c *ghostCache) put(key ghostCacheKey, ghost Ghost, present bool) {
	c.valid = true
	c.key = key
	c.present = present
	c.ghost = ghost
}

func (m *Model) ghostFor(row, col int, lineText string, rawLen int) (Ghost, bool) {
	if m.buf == nil || m.cfg.GhostProvider == nil || !m.focused {
		return Ghost{}, false
	}

	col = clampInt(col, 0, maxInt(rawLen, 0))

	key := ghostCacheKey{
		docID:      m.cfg.DocID,
		docVersion: m.buf.Version(),
		row:        row,
		col:        col,
	}
	if g, present := m.ghostCache.get(key); m.ghostCache.valid && m.ghostCache.key == key {
		return g, present
	}

	ctx := GhostContext{
		Row:         row,
		GraphemeCol: col,
		LineText:    lineText,
		IsEndOfLine: col == rawLen,
		DocID:       m.cfg.DocID,
		DocVersion:  m.buf.Version(),
	}

	ghost, present := m.cfg.GhostProvider(ctx)
	ghost.Text = sanitizeSingleLine(ghost.Text)
	if present && ghost.Text == "" && len(ghost.Edits) == 0 {
		present = false
	}
	m.ghostCache.put(key, ghost, present)
	return ghost, present
}

func (m *Model) virtualTextWithGhost(row int, rawLine string, vt VirtualText) VirtualText {
	if m.buf == nil || m.cfg.GhostProvider == nil || !m.focused {
		return vt
	}

	cursor := m.buf.Cursor()
	if cursor.Row != row {
		return vt
	}

	rawLen := graphemeutil.Count(rawLine)
	col := clampInt(cursor.GraphemeCol, 0, rawLen)

	ghost, ok := m.ghostFor(row, col, rawLine, rawLen)
	if !ok || ghost.Text == "" {
		return vt
	}

	vt.Insertions = append(vt.Insertions, VirtualInsertion{
		GraphemeCol: col,
		Text:        ghost.Text,
		Role:        VirtualRoleGhost,
		StyleKey:    ghost.StyleKey,
	})
	return normalizeVirtualText(vt, rawLen)
}
